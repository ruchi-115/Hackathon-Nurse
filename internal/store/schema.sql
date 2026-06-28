-- Raw tables mirror the mock PCC API. Primary keys make upserts idempotent so
-- reruns and incremental `since` syncs are safe.

CREATE TABLE IF NOT EXISTS patient (
    id                  INTEGER PRIMARY KEY,   -- internal id (notes/assessments)
    patient_id          TEXT NOT NULL,         -- external id (diagnoses/coverage)
    facility_id         INTEGER NOT NULL,
    first_name          TEXT,
    last_name           TEXT,
    birth_date          TEXT,
    gender              TEXT,
    primary_payer_code  TEXT,
    last_modified_at    TEXT,
    is_new_admission    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_patient_facility ON patient(facility_id);
CREATE INDEX IF NOT EXISTS idx_patient_extid ON patient(patient_id);

CREATE TABLE IF NOT EXISTS diagnosis (
    id                  INTEGER PRIMARY KEY,
    patient_id          TEXT NOT NULL,
    icd10_code          TEXT,
    icd10_description   TEXT,
    clinical_status     TEXT,
    onset_date          TEXT,
    last_modified_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_diagnosis_patient ON diagnosis(patient_id);

CREATE TABLE IF NOT EXISTS coverage (
    id                  INTEGER PRIMARY KEY,
    patient_id          TEXT NOT NULL,
    payer_name          TEXT,
    payer_code          TEXT,
    payer_type          TEXT,
    effective_from      TEXT,
    effective_to        TEXT,
    last_modified_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_coverage_patient ON coverage(patient_id);

CREATE TABLE IF NOT EXISTS note (
    id                  INTEGER PRIMARY KEY,
    patient_id          INTEGER NOT NULL,      -- internal id
    org_id              TEXT,
    pcc_note_id         INTEGER,
    note_type           TEXT,
    effective_date      TEXT,
    note_text           TEXT,
    created_by          TEXT,
    note_label          TEXT
);
CREATE INDEX IF NOT EXISTS idx_note_patient ON note(patient_id);

CREATE TABLE IF NOT EXISTS assessment (
    id                          INTEGER PRIMARY KEY,
    patient_id                  INTEGER NOT NULL,  -- internal id
    org_id                      TEXT,
    pcc_assessment_id           INTEGER,
    assessment_type             TEXT,
    status                      TEXT,
    assessment_date             TEXT,
    completion_date             TEXT,
    template_id                 INTEGER,
    assessment_type_description TEXT,
    raw_json                    TEXT
);
CREATE INDEX IF NOT EXISTS idx_assessment_patient ON assessment(patient_id);

-- Derived: one row per patient, the biller-facing output.
CREATE TABLE IF NOT EXISTS eligibility (
    patient_id          TEXT PRIMARY KEY,      -- external id
    internal_id         INTEGER NOT NULL,
    facility_id         INTEGER NOT NULL,
    has_active_mcb      INTEGER NOT NULL DEFAULT 0,
    wound_type          TEXT,
    stage               TEXT,
    location            TEXT,
    length_cm           REAL,
    width_cm            REAL,
    depth_cm            REAL,
    drainage_amount     TEXT,
    has_secondary_wound INTEGER NOT NULL DEFAULT 0,
    confidence          REAL NOT NULL DEFAULT 0,
    extraction_source   TEXT,
    decision            TEXT NOT NULL,
    reason              TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_eligibility_facility ON eligibility(facility_id);
CREATE INDEX IF NOT EXISTS idx_eligibility_decision ON eligibility(decision);
