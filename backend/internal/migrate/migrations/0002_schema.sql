CREATE TYPE agent_run_status AS ENUM ('CREATED', 'RUNNING', 'WAITING_USER', 'WAITING_CONFIRMATION', 'COMPLETED', 'ESCALATED', 'FAILED', 'CANCELLED');
CREATE TYPE agent_route AS ENUM ('KNOWLEDGE_ANSWER', 'ORDER_QUERY', 'TICKET_CREATION', 'CLARIFICATION', 'HUMAN_HANDOFF');
CREATE TYPE trace_status AS ENUM ('RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED');
CREATE TYPE knowledge_version_status AS ENUM ('DRAFT', 'PROCESSING', 'PENDING_REVIEW', 'PUBLISHED', 'FAILED', 'DISABLED');
CREATE TYPE knowledge_todo_status AS ENUM ('PENDING_SUPPLEMENT', 'PROCESSED', 'NO_ACTION_REQUIRED');
CREATE TYPE ticket_status AS ENUM ('PENDING', 'IN_PROGRESS', 'WAITING_CUSTOMER', 'RESOLVED', 'CLOSED', 'CANCELLED');
CREATE TYPE handoff_status AS ENUM ('WAITING', 'IN_PROGRESS', 'ENDED', 'CANCELLED');
CREATE TYPE durable_task_status AS ENUM ('PENDING', 'ENQUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED');
CREATE TYPE message_actor_type AS ENUM ('CUSTOMER', 'AGENT', 'MEMBER', 'SYSTEM');
CREATE TYPE message_content_type AS ENUM ('TEXT', 'ORDER_CARD', 'TICKET_CARD', 'HANDOFF_STATUS', 'SYSTEM_STATUS');
CREATE TYPE feedback_value AS ENUM ('HELPFUL', 'NOT_HELPFUL');
CREATE TYPE conversation_status AS ENUM ('OPEN', 'CLOSED');
CREATE TYPE response_owner AS ENUM ('AGENT', 'HUMAN');
CREATE TYPE tool_execution_status AS ENUM ('CREATED', 'VALIDATED', 'EXECUTING', 'SUCCEEDED', 'FAILED', 'UNKNOWN', 'CANCELLED');
CREATE TYPE tool_business_result AS ENUM ('SUCCESS', 'NOT_FOUND', 'INELIGIBLE', 'REJECTED', 'NOT_APPLICABLE', 'UNKNOWN');
CREATE TYPE ticket_priority AS ENUM ('LOW', 'NORMAL', 'HIGH', 'URGENT');
CREATE TYPE activity_visibility AS ENUM ('PUBLIC', 'INTERNAL');
CREATE TYPE knowledge_index_status AS ENUM ('BUILDING', 'ACTIVE', 'FAILED', 'RETIRED');
CREATE TYPE knowledge_build_status AS ENUM ('PENDING', 'PROCESSING', 'READY', 'FAILED', 'CANCELLED');
CREATE TYPE demo_session_status AS ENUM ('ACTIVE', 'EXPIRED', 'REVOKED', 'RESETTING', 'RESET');

CREATE TABLE tenants (
    id uuid PRIMARY KEY,
    slug text NOT NULL UNIQUE CHECK (slug = lower(slug) AND length(slug) BETWEEN 1 AND 64),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 160),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    is_demo boolean NOT NULL DEFAULT false,
    data_generation integer NOT NULL DEFAULT 1 CHECK (data_generation > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0)
);

CREATE TABLE demo_sessions (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    token_hash char(64) NOT NULL,
    session_type text NOT NULL CHECK (session_type IN ('VISITOR', 'DEVELOPER')),
    status demo_session_status NOT NULL DEFAULT 'ACTIVE',
    data_generation integer NOT NULL CHECK (data_generation > 0),
    request_limit integer NOT NULL CHECK (request_limit > 0),
    token_limit bigint NOT NULL CHECK (token_limit >= 0),
    upload_file_limit integer NOT NULL CHECK (upload_file_limit >= 0),
    request_count integer NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    token_count bigint NOT NULL DEFAULT 0 CHECK (token_count >= 0),
    upload_file_count integer NOT NULL DEFAULT 0 CHECK (upload_file_count >= 0),
    last_seen_at timestamptz,
    expires_at timestamptz NOT NULL,
    reset_requested_at timestamptz,
    reset_completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, token_hash),
    CHECK (expires_at > created_at)
);

CREATE TABLE customers (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    demo_session_id uuid,
    source text NOT NULL DEFAULT 'DEMO' CHECK (source IN ('DEMO', 'CONNECTOR')),
    external_subject_hash char(64),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 120),
    locale text NOT NULL DEFAULT 'zh-CN',
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED', 'EXPIRED')),
    expires_at timestamptz,
    anonymized_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, demo_session_id) REFERENCES demo_sessions(tenant_id, id)
);

CREATE UNIQUE INDEX customers_demo_session_uidx ON customers (tenant_id, demo_session_id) WHERE demo_session_id IS NOT NULL;
CREATE UNIQUE INDEX customers_external_subject_uidx ON customers (tenant_id, source, external_subject_hash) WHERE external_subject_hash IS NOT NULL;

CREATE TABLE workspace_members (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    subject_key text NOT NULL CHECK (length(subject_key) BETWEEN 1 AND 160),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 120),
    role text NOT NULL CHECK (role IN ('SUPPORT_AGENT', 'KNOWLEDGE_OPERATOR', 'DEMO_ADMIN')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, subject_key)
);

CREATE TABLE conversations (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    conversation_number text NOT NULL,
    customer_id uuid NOT NULL,
    status conversation_status NOT NULL DEFAULT 'OPEN',
    response_owner response_owner NOT NULL DEFAULT 'AGENT',
    subject text CHECK (subject IS NULL OR length(subject) <= 240),
    consecutive_unresolved_count smallint NOT NULL DEFAULT 0 CHECK (consecutive_unresolved_count BETWEEN 0 AND 3),
    resolution_outcome text,
    human_involved boolean NOT NULL DEFAULT false,
    next_message_sequence bigint NOT NULL DEFAULT 1 CHECK (next_message_sequence > 0),
    last_message_at timestamptz,
    closed_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, conversation_number),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    CHECK ((status = 'CLOSED') = (closed_at IS NOT NULL))
);

CREATE TABLE messages (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    conversation_id uuid NOT NULL,
    sequence_no bigint NOT NULL CHECK (sequence_no > 0),
    actor_type message_actor_type NOT NULL,
    customer_id uuid,
    member_id uuid,
    agent_run_id uuid,
    content_type message_content_type NOT NULL,
    content_text text CHECK (content_text IS NULL OR length(content_text) <= 50000),
    content_payload jsonb,
    locale text,
    redaction_state text NOT NULL DEFAULT 'APPLIED' CHECK (redaction_state IN ('APPLIED', 'NOT_REQUIRED')),
    content_sha256 char(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, conversation_id, sequence_no),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, member_id) REFERENCES workspace_members(tenant_id, id),
    CHECK (content_text IS NOT NULL OR content_payload IS NOT NULL),
    CHECK ((actor_type = 'CUSTOMER' AND customer_id IS NOT NULL AND member_id IS NULL) OR
           (actor_type = 'MEMBER' AND member_id IS NOT NULL AND customer_id IS NULL) OR
           (actor_type IN ('AGENT', 'SYSTEM') AND customer_id IS NULL AND member_id IS NULL))
);

CREATE TABLE message_feedback (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    message_id uuid NOT NULL,
    conversation_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    value feedback_value NOT NULL,
    reason_code text CHECK (reason_code IS NULL OR length(reason_code) <= 64),
    comment_summary text CHECK (comment_summary IS NULL OR length(comment_summary) <= 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, message_id, customer_id),
    FOREIGN KEY (tenant_id, message_id) REFERENCES messages(tenant_id, id),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id)
);

CREATE TABLE agent_runs (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    conversation_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    trigger_message_id uuid NOT NULL,
    output_message_id uuid,
    client_request_id text NOT NULL CHECK (length(client_request_id) BETWEEN 1 AND 128),
    status agent_run_status NOT NULL DEFAULT 'CREATED',
    current_route agent_route,
    route_reason_code text,
    step_count smallint NOT NULL DEFAULT 0 CHECK (step_count >= 0),
    max_steps smallint NOT NULL CHECK (max_steps BETWEEN 1 AND 20),
    retry_count smallint NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    max_retries smallint NOT NULL CHECK (max_retries BETWEEN 0 AND 3),
    pending_action_type text,
    pending_action_summary jsonb,
    confirmation_expires_at timestamptz,
    failure_code text,
    failure_summary text CHECK (failure_summary IS NULL OR length(failure_summary) <= 500),
    input_tokens bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    estimated_cost_micros bigint NOT NULL DEFAULT 0 CHECK (estimated_cost_micros >= 0),
    started_at timestamptz,
    heartbeat_at timestamptz,
    waiting_since timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, conversation_id, client_request_id),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, trigger_message_id) REFERENCES messages(tenant_id, id)
);

CREATE UNIQUE INDEX agent_runs_active_uidx ON agent_runs (tenant_id, conversation_id) WHERE status IN ('CREATED', 'RUNNING', 'WAITING_USER', 'WAITING_CONFIRMATION');

CREATE TABLE agent_traces (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    agent_run_id uuid NOT NULL,
    status trace_status NOT NULL DEFAULT 'RUNNING',
    schema_version smallint NOT NULL DEFAULT 1 CHECK (schema_version > 0),
    redaction_policy_version text NOT NULL,
    event_count integer NOT NULL DEFAULT 0 CHECK (event_count >= 0),
    total_duration_ms bigint CHECK (total_duration_ms IS NULL OR total_duration_ms >= 0),
    started_at timestamptz NOT NULL DEFAULT now(),
    ended_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, agent_run_id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id)
);

CREATE TABLE agent_trace_events (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    trace_id uuid,
    agent_run_id uuid NOT NULL,
    sequence_no integer NOT NULL CHECK (sequence_no > 0),
    stage text NOT NULL CHECK (stage IN ('RUN', 'ROUTING', 'POLICY', 'RETRIEVAL', 'MODEL', 'TOOL', 'HANDOFF', 'RESPONSE')),
    event_type text NOT NULL CHECK (length(event_type) BETWEEN 1 AND 80),
    event_status text NOT NULL CHECK (event_status IN ('STARTED', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    route agent_route,
    reason_code text,
    message_id uuid,
    tool_call_id uuid,
    citation_id uuid,
    safe_summary text CHECK (safe_summary IS NULL OR length(safe_summary) <= 500),
    safe_metadata jsonb,
    duration_ms integer CHECK (duration_ms IS NULL OR duration_ms >= 0),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, trace_id, sequence_no),
    FOREIGN KEY (tenant_id, trace_id) REFERENCES agent_traces(tenant_id, id) ON DELETE SET NULL,
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, message_id) REFERENCES messages(tenant_id, id)
);

CREATE TABLE model_usage_records (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    agent_run_id uuid NOT NULL,
    trace_id uuid NOT NULL,
    stage text NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    status text NOT NULL CHECK (status IN ('SUCCEEDED', 'FAILED', 'CANCELLED')),
    input_tokens bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    estimated_cost_micros bigint NOT NULL DEFAULT 0 CHECK (estimated_cost_micros >= 0),
    usage_source text NOT NULL CHECK (usage_source IN ('PROVIDER', 'ESTIMATED', 'UNAVAILABLE')),
    latency_ms integer NOT NULL CHECK (latency_ms >= 0),
    error_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, trace_id) REFERENCES agent_traces(tenant_id, id)
);

CREATE TABLE knowledge_documents (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    document_key text NOT NULL CHECK (length(document_key) BETWEEN 1 AND 128),
    title text NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
    product_key text,
    category text,
    owner_demo_session_id uuid,
    is_disabled boolean NOT NULL DEFAULT false,
    disabled_at timestamptz,
    created_by_member_id uuid,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, document_key),
    FOREIGN KEY (tenant_id, owner_demo_session_id) REFERENCES demo_sessions(tenant_id, id),
    FOREIGN KEY (tenant_id, created_by_member_id) REFERENCES workspace_members(tenant_id, id)
);

CREATE TABLE knowledge_document_versions (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    version_no integer NOT NULL CHECK (version_no > 0),
    status knowledge_version_status NOT NULL DEFAULT 'DRAFT',
    is_current_published boolean NOT NULL DEFAULT false,
    source_type text NOT NULL CHECK (source_type IN ('MARKDOWN', 'TEXT_PDF')),
    source_filename text NOT NULL CHECK (length(source_filename) BETWEEN 1 AND 255),
    media_type text NOT NULL,
    object_key text NOT NULL,
    byte_size bigint NOT NULL CHECK (byte_size BETWEEN 1 AND 10485760),
    content_sha256 char(64) NOT NULL,
    parser_version text,
    normalized_content_sha256 char(64),
    created_by_member_id uuid,
    reviewed_by_member_id uuid,
    review_note text CHECK (review_note IS NULL OR length(review_note) <= 1000),
    failure_code text,
    failure_summary text CHECK (failure_summary IS NULL OR length(failure_summary) <= 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, document_id, version_no),
    FOREIGN KEY (tenant_id, document_id) REFERENCES knowledge_documents(tenant_id, id),
    FOREIGN KEY (tenant_id, created_by_member_id) REFERENCES workspace_members(tenant_id, id),
    FOREIGN KEY (tenant_id, reviewed_by_member_id) REFERENCES workspace_members(tenant_id, id)
);

CREATE UNIQUE INDEX knowledge_published_uidx ON knowledge_document_versions (tenant_id, document_id) WHERE is_current_published;

CREATE TABLE knowledge_index_versions (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    version_no integer NOT NULL CHECK (version_no > 0),
    status knowledge_index_status NOT NULL DEFAULT 'BUILDING',
    embedding_provider text NOT NULL,
    embedding_model text NOT NULL,
    vector_dimension smallint NOT NULL CHECK (vector_dimension BETWEEN 128 AND 4096),
    distance_metric text NOT NULL DEFAULT 'COSINE' CHECK (distance_metric = 'COSINE'),
    chunker_version text NOT NULL,
    normalizer_version text NOT NULL,
    lexical_config text NOT NULL,
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    activated_at timestamptz,
    retired_at timestamptz,
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, version_no)
);

CREATE UNIQUE INDEX knowledge_active_index_uidx ON knowledge_index_versions (tenant_id) WHERE status = 'ACTIVE';

CREATE TABLE knowledge_index_builds (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    document_version_id uuid NOT NULL,
    index_version_id uuid NOT NULL,
    durable_task_id uuid,
    status knowledge_build_status NOT NULL DEFAULT 'PENDING',
    chunk_count integer NOT NULL DEFAULT 0 CHECK (chunk_count >= 0),
    build_sha256 char(64),
    failure_code text,
    failure_summary text CHECK (failure_summary IS NULL OR length(failure_summary) <= 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, document_id) REFERENCES knowledge_documents(tenant_id, id),
    FOREIGN KEY (tenant_id, document_version_id) REFERENCES knowledge_document_versions(tenant_id, id),
    FOREIGN KEY (tenant_id, index_version_id) REFERENCES knowledge_index_versions(tenant_id, id)
);

CREATE TABLE knowledge_chunks (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    document_version_id uuid NOT NULL,
    index_version_id uuid NOT NULL,
    index_build_id uuid NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    section_title text,
    section_path text[] NOT NULL DEFAULT '{}',
    page_number integer CHECK (page_number IS NULL OR page_number > 0),
    normalized_text text NOT NULL CHECK (length(normalized_text) BETWEEN 1 AND 16000),
    content_sha256 char(64) NOT NULL,
    token_count integer NOT NULL CHECK (token_count > 0),
    search_vector tsvector NOT NULL,
    embedding_dimension smallint NOT NULL CHECK (embedding_dimension > 0),
    embedding vector(128) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, index_build_id, ordinal),
    FOREIGN KEY (tenant_id, document_id) REFERENCES knowledge_documents(tenant_id, id),
    FOREIGN KEY (tenant_id, document_version_id) REFERENCES knowledge_document_versions(tenant_id, id),
    FOREIGN KEY (tenant_id, index_version_id) REFERENCES knowledge_index_versions(tenant_id, id),
    FOREIGN KEY (tenant_id, index_build_id) REFERENCES knowledge_index_builds(tenant_id, id)
);

CREATE INDEX knowledge_chunks_search_vector_idx ON knowledge_chunks USING GIN (search_vector);
CREATE INDEX knowledge_chunks_embedding_idx ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

CREATE TABLE citations (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    agent_run_id uuid NOT NULL,
    assistant_message_id uuid NOT NULL,
    document_id uuid NOT NULL,
    document_version_id uuid NOT NULL,
    index_version_id uuid NOT NULL,
    chunk_id uuid NOT NULL,
    rank smallint NOT NULL CHECK (rank > 0),
    source_title text NOT NULL,
    section_title text,
    page_number integer CHECK (page_number IS NULL OR page_number > 0),
    quote_excerpt text CHECK (quote_excerpt IS NULL OR length(quote_excerpt) <= 500),
    lexical_score double precision,
    vector_score double precision,
    fused_score double precision NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, assistant_message_id, rank),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, assistant_message_id) REFERENCES messages(tenant_id, id),
    FOREIGN KEY (tenant_id, document_id) REFERENCES knowledge_documents(tenant_id, id),
    FOREIGN KEY (tenant_id, document_version_id) REFERENCES knowledge_document_versions(tenant_id, id),
    FOREIGN KEY (tenant_id, index_version_id) REFERENCES knowledge_index_versions(tenant_id, id),
    FOREIGN KEY (tenant_id, chunk_id) REFERENCES knowledge_chunks(tenant_id, id)
);

CREATE TABLE knowledge_operations_todos (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    status knowledge_todo_status NOT NULL DEFAULT 'PENDING_SUPPLEMENT',
    trigger_reason text NOT NULL,
    dedupe_key char(64) NOT NULL,
    conversation_id uuid NOT NULL,
    agent_run_id uuid,
    message_id uuid,
    feedback_id uuid,
    primary_citation_id uuid,
    route agent_route,
    question_summary text NOT NULL CHECK (length(question_summary) BETWEEN 1 AND 500),
    assigned_member_id uuid,
    resolution_code text,
    resolution_note text CHECK (resolution_note IS NULL OR length(resolution_note) <= 1000),
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, dedupe_key),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, message_id) REFERENCES messages(tenant_id, id),
    FOREIGN KEY (tenant_id, feedback_id) REFERENCES message_feedback(tenant_id, id),
    FOREIGN KEY (tenant_id, primary_citation_id) REFERENCES citations(tenant_id, id),
    FOREIGN KEY (tenant_id, assigned_member_id) REFERENCES workspace_members(tenant_id, id)
);

CREATE TABLE mock_orders (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    customer_id uuid NOT NULL,
    order_number text NOT NULL,
    status text NOT NULL,
    purchased_at timestamptz NOT NULL,
    warranty_valid_until timestamptz,
    currency char(3) NOT NULL,
    total_amount_minor bigint NOT NULL CHECK (total_amount_minor >= 0),
    data_generation integer NOT NULL CHECK (data_generation > 0),
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, customer_id, order_number),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id)
);

CREATE TABLE mock_order_items (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    order_id uuid NOT NULL,
    product_key text NOT NULL,
    sku text NOT NULL,
    product_name text NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_amount_minor bigint NOT NULL CHECK (unit_amount_minor >= 0),
    warranty_valid_until timestamptz,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, order_id, sku),
    FOREIGN KEY (tenant_id, order_id) REFERENCES mock_orders(tenant_id, id)
);

CREATE TABLE tickets (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    ticket_number text NOT NULL,
    customer_id uuid NOT NULL,
    conversation_id uuid,
    agent_run_id uuid,
    mock_order_id uuid,
    mock_order_item_id uuid,
    order_reference_snapshot text,
    product_reference_snapshot text,
    problem_type text NOT NULL,
    problem_summary text NOT NULL CHECK (length(problem_summary) BETWEEN 1 AND 2000),
    problem_fingerprint char(64) NOT NULL,
    duplicate_scope_hash char(64) NOT NULL,
    priority ticket_priority NOT NULL DEFAULT 'NORMAL',
    status ticket_status NOT NULL DEFAULT 'PENDING',
    source text NOT NULL CHECK (source IN ('AGENT', 'HUMAN', 'DEMO_SEED')),
    creation_reason text NOT NULL,
    eligibility_result text,
    eligibility_checked_at timestamptz,
    assignee_member_id uuid,
    next_activity_sequence integer NOT NULL DEFAULT 1 CHECK (next_activity_sequence > 0),
    claimed_at timestamptz,
    resolved_at timestamptz,
    closed_at timestamptz,
    cancelled_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, ticket_number),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, mock_order_id) REFERENCES mock_orders(tenant_id, id),
    FOREIGN KEY (tenant_id, mock_order_item_id) REFERENCES mock_order_items(tenant_id, id),
    FOREIGN KEY (tenant_id, assignee_member_id) REFERENCES workspace_members(tenant_id, id)
);

CREATE UNIQUE INDEX tickets_duplicate_active_uidx ON tickets (tenant_id, duplicate_scope_hash) WHERE status IN ('PENDING', 'IN_PROGRESS', 'WAITING_CUSTOMER', 'RESOLVED');

CREATE TABLE ticket_activities (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    ticket_id uuid NOT NULL,
    sequence_no integer NOT NULL CHECK (sequence_no > 0),
    activity_type text NOT NULL,
    visibility activity_visibility NOT NULL,
    actor_type message_actor_type NOT NULL,
    customer_id uuid,
    member_id uuid,
    agent_run_id uuid,
    from_status ticket_status,
    to_status ticket_status,
    from_assignee_id uuid,
    to_assignee_id uuid,
    body text CHECK (body IS NULL OR length(body) <= 5000),
    safe_metadata jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, ticket_id, sequence_no),
    FOREIGN KEY (tenant_id, ticket_id) REFERENCES tickets(tenant_id, id),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, member_id) REFERENCES workspace_members(tenant_id, id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id)
);

CREATE TABLE handoff_requests (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    handoff_number text NOT NULL,
    conversation_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    agent_run_id uuid,
    status handoff_status NOT NULL DEFAULT 'WAITING',
    reason_code text NOT NULL,
    context_summary text NOT NULL CHECK (length(context_summary) BETWEEN 1 AND 2000),
    knowledge_gap boolean NOT NULL DEFAULT false,
    priority ticket_priority NOT NULL DEFAULT 'NORMAL',
    assigned_member_id uuid,
    requested_at timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    ended_at timestamptz,
    cancelled_at timestamptz,
    wait_duration_ms bigint CHECK (wait_duration_ms IS NULL OR wait_duration_ms >= 0),
    sla_due_at timestamptz,
    outcome_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, handoff_number),
    FOREIGN KEY (tenant_id, conversation_id) REFERENCES conversations(tenant_id, id),
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id),
    FOREIGN KEY (tenant_id, assigned_member_id) REFERENCES workspace_members(tenant_id, id)
);

CREATE UNIQUE INDEX handoff_active_uidx ON handoff_requests (tenant_id, conversation_id) WHERE status IN ('WAITING', 'IN_PROGRESS');

CREATE TABLE notifications (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    recipient_type text NOT NULL CHECK (recipient_type IN ('CUSTOMER', 'MEMBER')),
    recipient_customer_id uuid,
    recipient_member_id uuid,
    event_type text NOT NULL,
    title_code text NOT NULL,
    title_params jsonb NOT NULL DEFAULT '{}'::jsonb,
    target_type text NOT NULL CHECK (target_type IN ('CONVERSATION', 'TICKET', 'HANDOFF', 'KNOWLEDGE_TODO')),
    target_id uuid NOT NULL,
    navigation_path text NOT NULL CHECK (navigation_path LIKE '/%'),
    read_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, recipient_customer_id) REFERENCES customers(tenant_id, id),
    FOREIGN KEY (tenant_id, recipient_member_id) REFERENCES workspace_members(tenant_id, id),
    CHECK ((recipient_type = 'CUSTOMER' AND recipient_customer_id IS NOT NULL AND recipient_member_id IS NULL) OR
           (recipient_type = 'MEMBER' AND recipient_member_id IS NOT NULL AND recipient_customer_id IS NULL))
);

CREATE TABLE durable_tasks (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    id uuid NOT NULL,
    task_type text NOT NULL,
    dedupe_key text NOT NULL CHECK (length(dedupe_key) BETWEEN 1 AND 200),
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    payload_version smallint NOT NULL DEFAULT 1 CHECK (payload_version > 0),
    safe_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status durable_task_status NOT NULL DEFAULT 'PENDING',
    priority smallint NOT NULL DEFAULT 0,
    attempt_count smallint NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts smallint NOT NULL CHECK (max_attempts > 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    asynq_task_id text,
    lease_owner text,
    leased_until timestamptz,
    last_error_code text,
    last_error_summary text CHECK (last_error_summary IS NULL OR length(last_error_summary) <= 500),
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    row_version bigint NOT NULL DEFAULT 1 CHECK (row_version > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, task_type, dedupe_key),
    CHECK (attempt_count <= max_attempts)
);

ALTER TABLE messages ADD CONSTRAINT messages_agent_run_fk FOREIGN KEY (tenant_id, agent_run_id) REFERENCES agent_runs(tenant_id, id);
ALTER TABLE agent_runs ADD CONSTRAINT agent_runs_output_message_fk FOREIGN KEY (tenant_id, output_message_id) REFERENCES messages(tenant_id, id);

CREATE INDEX demo_sessions_expiry_idx ON demo_sessions (tenant_id, status, expires_at);
CREATE INDEX workspace_members_role_idx ON workspace_members (tenant_id, role, status);
CREATE INDEX conversations_customer_idx ON conversations (tenant_id, customer_id, last_message_at DESC);
CREATE INDEX messages_agent_run_idx ON messages (tenant_id, agent_run_id);
CREATE INDEX agent_runs_status_idx ON agent_runs (tenant_id, status, heartbeat_at);
CREATE INDEX agent_traces_status_idx ON agent_traces (tenant_id, status, created_at DESC);
CREATE INDEX agent_trace_events_run_idx ON agent_trace_events (tenant_id, agent_run_id, sequence_no);
CREATE INDEX model_usage_run_idx ON model_usage_records (tenant_id, agent_run_id, created_at);
CREATE INDEX knowledge_documents_scope_idx ON knowledge_documents (tenant_id, is_disabled, category, product_key);
CREATE INDEX knowledge_build_status_idx ON knowledge_index_builds (tenant_id, status, updated_at);
CREATE INDEX knowledge_todos_status_idx ON knowledge_operations_todos (tenant_id, status, created_at);
CREATE INDEX mock_orders_customer_idx ON mock_orders (tenant_id, customer_id, purchased_at DESC);
CREATE INDEX tickets_queue_idx ON tickets (tenant_id, status, priority DESC, created_at);
CREATE INDEX tickets_customer_idx ON tickets (tenant_id, customer_id, created_at DESC);
CREATE INDEX ticket_activities_ticket_idx ON ticket_activities (tenant_id, ticket_id, sequence_no);
CREATE INDEX handoff_queue_idx ON handoff_requests (tenant_id, status, priority DESC, requested_at);
CREATE INDEX notifications_customer_unread_idx ON notifications (tenant_id, recipient_customer_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX durable_tasks_queue_idx ON durable_tasks (tenant_id, status, available_at);

INSERT INTO tenants (id, slug, display_name, is_demo)
VALUES ('00000000-0000-0000-0000-000000000001', 'novatech', 'NovaTech 电子商城', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO workspace_members (tenant_id, id, subject_key, display_name, role)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000004', 'demo-support-agent', 'NovaTech 客服', 'SUPPORT_AGENT'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000005', 'demo-knowledge-operator', 'NovaTech 知识运营', 'KNOWLEDGE_OPERATOR')
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO demo_sessions (tenant_id, id, token_hash, session_type, data_generation, request_limit, token_limit, upload_file_limit, expires_at)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000008', repeat('b', 64), 'DEVELOPER', 1, 1000, 1000000, 20, now() + interval '365 days')
ON CONFLICT DO NOTHING;

INSERT INTO demo_sessions (tenant_id, id, token_hash, session_type, data_generation, request_limit, token_limit, upload_file_limit, expires_at)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', repeat('a', 64), 'VISITOR', 1, 100, 100000, 5, now() + interval '24 hours')
ON CONFLICT DO NOTHING;

INSERT INTO customers (tenant_id, id, demo_session_id, display_name, locale)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000002', '演示客户', 'zh-CN')
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO mock_orders (tenant_id, id, customer_id, order_number, status, purchased_at, warranty_valid_until, currency, total_amount_minor, data_generation)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000003', 'SF20260001', 'PAID', '2026-07-01T00:00:00Z', '2027-07-01T00:00:00Z', 'CNY', 29900, 1)
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO mock_order_items (tenant_id, id, order_id, product_key, sku, product_name, quantity, unit_amount_minor, warranty_valid_until, attributes)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000006', 'bluetooth-earbuds', 'SF-EARBUDS-001', 'NovaTech 蓝牙耳机', 1, 29900, '2027-07-01T00:00:00Z', '{"color":"黑色"}'::jsonb)
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO knowledge_documents (tenant_id, id, document_key, title, product_key, category)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', 'novatech-earbuds-manual', '蓝牙耳机产品手册', 'bluetooth-earbuds', '产品手册'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000011', 'novatech-after-sales-policy', '售后政策', NULL, '售后政策'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000012', 'novatech-earbuds-faq', '蓝牙耳机常见问题 FAQ', 'bluetooth-earbuds', '常见问题')
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO knowledge_document_versions (tenant_id, id, document_id, version_no, status, is_current_published, source_type, source_filename, media_type, object_key, byte_size, content_sha256, parser_version, normalized_content_sha256)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000010', 1, 'PUBLISHED', true, 'MARKDOWN', '蓝牙耳机产品手册.md', 'text/markdown', 'demo/novatech-earbuds-manual-v1.md', 1024, repeat('1', 64), 'markdown-v1', repeat('2', 64)),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000011', 1, 'PUBLISHED', true, 'MARKDOWN', '售后政策.md', 'text/markdown', 'demo/novatech-after-sales-policy-v1.md', 1024, repeat('3', 64), 'markdown-v1', repeat('4', 64)),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000012', 1, 'PUBLISHED', true, 'MARKDOWN', '蓝牙耳机FAQ.md', 'text/markdown', 'demo/novatech-earbuds-faq-v1.md', 1024, repeat('5', 64), 'markdown-v1', repeat('6', 64))
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO knowledge_index_versions (tenant_id, id, version_no, status, embedding_provider, embedding_model, vector_dimension, chunker_version, normalizer_version, lexical_config, activated_at)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000016', 1, 'ACTIVE', 'mock', 'mock-v1', 128, 'markdown-v1', 'default-v1', 'simple', now())
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO knowledge_index_builds (tenant_id, id, document_id, document_version_id, index_version_id, status, chunk_count, build_sha256)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000017', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000016', 'READY', 1, repeat('7', 64)),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000018', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000016', 'READY', 1, repeat('8', 64)),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000019', '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000016', 'READY', 1, repeat('9', 64))
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO knowledge_chunks (tenant_id, id, document_id, document_version_id, index_version_id, index_build_id, ordinal, section_title, section_path, normalized_text, content_sha256, token_count, search_vector, embedding_dimension, embedding)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000017', 1, '单耳无声排查', ARRAY['故障排查', '单耳无声'], '请先将双耳机放回充电盒，等待十秒后取出并重新连接蓝牙；如果仍无声音，请尝试恢复出厂设置。', repeat('a', 64), 32, to_tsvector('simple', '单耳无声 重置 蓝牙'), 128, ('[' || repeat('0,', 127) || '0]')::vector),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000018', 1, '保修规则', ARRAY['售后政策', '保修'], '产品自购买日起享有十二个月有限保修，符合保修条件的产品可以申请售后处理。', repeat('b', 64), 24, to_tsvector('simple', '保修 十二个月 售后'), 128, ('[' || repeat('0,', 127) || '0]')::vector),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000019', 1, '常见问题', ARRAY['FAQ', '连接'], '蓝牙连接异常时，可以先关闭手机蓝牙再重新开启，并确认耳机没有连接到其他设备。', repeat('c', 64), 28, to_tsvector('simple', '蓝牙 连接 异常'), 128, ('[' || repeat('0,', 127) || '0]')::vector)
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO conversations (tenant_id, id, conversation_number, customer_id, subject)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000030', 'CV20260001', '00000000-0000-0000-0000-000000000003', '蓝牙耳机售后咨询')
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO messages (tenant_id, id, conversation_id, sequence_no, actor_type, customer_id, content_type, content_text, locale, content_sha256)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000031', '00000000-0000-0000-0000-000000000030', 1, 'CUSTOMER', '00000000-0000-0000-0000-000000000003', 'TEXT', '我的蓝牙耳机左耳没有声音了', 'zh-CN', repeat('d', 64))
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO tickets (tenant_id, id, ticket_number, customer_id, conversation_id, mock_order_id, mock_order_item_id, order_reference_snapshot, product_reference_snapshot, problem_type, problem_summary, problem_fingerprint, duplicate_scope_hash, source, creation_reason, eligibility_result, eligibility_checked_at)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000032', 'TK20260001', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000007', 'SF20260001', 'NovaTech 蓝牙耳机', 'PRODUCT_FAULT', '左耳无声音，重置和重新连接后仍未恢复', repeat('e', 64), repeat('f', 64), 'DEMO_SEED', 'DEMO_SEED', 'ELIGIBLE', now())
ON CONFLICT (tenant_id, id) DO NOTHING;

INSERT INTO ticket_activities (tenant_id, id, ticket_id, sequence_no, activity_type, visibility, actor_type, body, safe_metadata)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000033', '00000000-0000-0000-0000-000000000032', 1, 'CREATED', 'PUBLIC', 'SYSTEM', '已创建售后工单。', '{"source":"DEMO_SEED"}'::jsonb)
ON CONFLICT (tenant_id, id) DO NOTHING;
