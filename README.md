# Resell Marketplace APIs

A high-performance backend API for a peer-to-peer secondhand marketplace built with **Go**, **PostgreSQL**, and **Cloudflare R2**.

---

## Image Upload & Worker Architecture

The application implements an asynchronous **Transactional Outbox Pattern** with a background worker engine to handle image uploads, validation, normalization, and lifecycle management without blocking user requests.

![Image Upload Architecture](docs/architecture.png)

<details>
<summary><b>Click to view interactive Mermaid flowchart</b></summary>

```mermaid
flowchart TD
    User(["User: POST /listings"]) -->|Creates listing + job| CreateJob["Insert into jobs table\n(status: processing)"]
    
    subgraph WorkerEngine ["Worker Processing Loop"]
        Claim["Worker Claims Job\n(FOR UPDATE SKIP LOCKED)"]
        
        CreateJob -.-> Claim
        
        Claim --> Fetch["Fetch uploads/... from R2"]
        
        Fetch --> ObjExists{"Object\nExists?"}
        
        %% Missing image branch
        ObjExists -- "NO" --> Missing["Gone or Never Uploaded"]
        Missing --> TxRejectBegin["BEGIN DB Transaction"]
        TxRejectBegin --> UpdateReject["UPDATE listings SET status = 'rejected'"]
        UpdateReject --> DeleteJob1["DELETE FROM jobs WHERE id = $1"]
        DeleteJob1 --> TxRejectCommit["COMMIT Transaction"]
        TxRejectCommit --> Next1["Straight to Next Job"]
        Next1 --> Claim
        
        %% Object exists branch
        ObjExists -- "YES" --> Normalize["Normalize to JPEG\n(Stream, Strip EXIF, Downscale)"]
        
        Normalize -- "Permanent Fail (Corrupt/Decode)" --> DLQ["Dead Letter Queue\n(Listing rejected & job deleted)"]
        DLQ --> Next2["Straight to Next Job"]
        Next2 --> Claim
        
        Normalize -- "Success" --> PutR2["PUT listings/... to R2\n(Promote normalized images)"]
        
        PutR2 -- "Temporary Fail" --> UpdateRetry["UPDATE jobs\nSET last_error = $1\n(Exponential run_after backoff)"]
        UpdateRetry -.->|"Retried when run_after <= NOW()"| Claim
        
        PutR2 -- "Success" --> TxReadyBegin["BEGIN DB Transaction"]
        TxReadyBegin --> UpdateReady["UPDATE listings SET status = 'ready'"]
        UpdateReady --> DeleteJob2["DELETE FROM jobs WHERE id = $1"]
        DeleteJob2 --> TxReadyCommit["COMMIT Transaction"]
        TxReadyCommit --> Next3["Straight to Next Job"]
        Next3 --> Claim
    end

    classDef success fill:#064e3b,stroke:#10b981,stroke-width:2px,color:#fff;
    classDef fail fill:#7f1d1d,stroke:#ef4444,stroke-width:2px,color:#fff;
    classDef process fill:#1e293b,stroke:#38bdf8,stroke-width:2px,color:#fff;
    classDef tx fill:#4c1d95,stroke:#8b5cf6,stroke-width:2px,color:#fff;
    
    class UpdateReady,TxReadyCommit success;
    class UpdateReject,DLQ,UpdateRetry,TxRejectCommit fail;
    class Claim,Fetch,Normalize,PutR2 process;
    class TxRejectBegin,TxReadyBegin tx;
```

</details>

---

### Step-by-Step Architecture Breakdown

#### 1. Upload & Job Creation (`POST /listings`)
1. Client requests presigned upload URLs for raw user uploads (`uploads/<user-id>/<uuid>.<ext>`).
2. Client uploads images directly to Cloudflare R2 via presigned PUT.
3. Client submits `POST /listings`.
4. Inside a single database transaction:
   * Listing record is inserted with `status = 'processing'`.
   * Image metadata records are inserted into `images`.
   * An image processing job is queued into `jobs` table (`kind = 'process_listing_images'`).

#### 2. Worker Claiming (`FOR UPDATE SKIP LOCKED`)
* Multiple workers concurrently poll the `jobs` queue without race conditions or head-of-line blocking using:
  ```sql
  UPDATE jobs 
  SET attempts = attempts + 1,
      run_after = NOW() + LEAST(INTERVAL '30 seconds' * POWER(2, attempts), INTERVAL '1 hour')
  WHERE id = (
      SELECT id FROM jobs 
      WHERE run_after <= NOW() AND attempts < 4
      ORDER BY run_after, id
      FOR UPDATE SKIP LOCKED 
      LIMIT 1
  )
  RETURNING id, kind, payload, attempts;
  ```

#### 3. Image Verification & Streamed Normalization
* **Fetch (`uploads/...`)**:
  * If object is missing (`storage.ErrNotFound`): The upload was abandoned. An atomic transaction updates listing `status = 'rejected'` and deletes the job.
* **Normalize to JPEG**:
  * Streams raw image bytes directly into `imaging.Normalize` (avoids buffering all images simultaneously in RAM).
  * Strips EXIF metadata to protect user privacy.
  * Downscales images to max bounding edge (e.g. 2048px) maintaining aspect ratio.
  * Re-encodes cleanly to JPEG with quality 85.
  * **Permanent failure**: If decoding fails (corrupted file/payload), job is routed to `deadLetterJob` to reject the listing immediately.

#### 4. Promotion to Permanent Storage (`listings/...`)
* Normalized images are uploaded to their permanent destination (`listings/<listing-id>/<uuid>.jpeg`).
* **Transient failure**: If R2 upload fails (e.g. network blip), the job updates `last_error` and is scheduled for exponential backoff retry via `run_after`.

#### 5. Completion Transaction
* Upon successful promotion of all images:
  * Database transaction opens.
  * Listing status is updated to `status = 'ready'`.
  * Job is removed from the `jobs` table.
  * Transaction commits atomically.
  * Worker immediately proceeds to claim the next job.
