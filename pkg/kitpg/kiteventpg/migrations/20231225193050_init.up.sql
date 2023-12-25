create table if not exists kitevent.events
(
    id         serial primary key,
    payload    jsonb,
    event_name varchar(255),
    created_at timestamp without time zone default (now() at time zone 'utc'),
    updated_at timestamp without time zone default (now() at time zone 'utc')
);

create table if not exists kitevent.event_processing_states
(
    id                                serial primary key,
    consumer_name                     varchar(255),
    event_id                          int,
    status                            varchar(255),
    error                             varchar(255),
    consumer_option_max_retries       int,
    consumer_option_retry_interval_ms int,
    consumer_option_timeout_ms        int,
    created_at                        timestamp without time zone default (now() at time zone 'utc'),
    updated_at                        timestamp without time zone default (now() at time zone 'utc'),
    processable_at                    timestamp without time zone default (now() at time zone 'utc'),
    run_at                            timestamp without time zone default (now() at time zone 'utc'),
    retry_number                      int,
    duration_ms                       int,
    timeout_at                        timestamp without time zone GENERATED ALWAYS AS (coalesce(pending_at, created_at) +
                                                                                       (consumer_option_timeout_ms * interval '1 millisecond')) STORED,
    failed_at                         timestamp without time zone default (now() at time zone 'utc'),
    success_at                        timestamp without time zone default (now() at time zone 'utc'),
    pending_at                        timestamp without time zone default (now() at time zone 'utc')
);

create index if not exists event_processing_states_timeout_at_idx on kitevent.event_processing_states (timeout_at);
create index if not exists event_processing_states_status_idx on kitevent.event_processing_states (status);
