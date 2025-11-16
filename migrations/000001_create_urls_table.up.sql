create table urls (
    id serial primary key,
    original varchar(255) not null,
    short varchar(255) not null
);

create index idx_urls_short on urls(short);