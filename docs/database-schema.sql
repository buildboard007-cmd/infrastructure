-- BuildBoard Infrastructure Database Schema
-- Single Multi-Schema PostgreSQL Database for Construction Management System
-- Schemas: iam (Identity and Access Management), project (Construction Management)

-- IAM Schema Tables (Identity and Access Management)

create table organizations
(
    id             bigserial
        primary key,
    name           varchar(255)                                           not null,
    org_type       varchar(50)                                            not null
        constraint organizations_org_type_check
            check ((org_type)::text = ANY
                   ((ARRAY ['general_contractor'::character varying, 'subcontractor'::character varying, 'architect'::character varying, 'owner'::character varying, 'consultant'::character varying])::text[])),
    license_number varchar(100),
    address        text,
    phone          varchar(20),
    email          varchar(255),
    website        varchar(255),
    status         varchar(50) default 'pending_setup'::character varying not null
        constraint organizations_status_check
            check ((status)::text = ANY
                   ((ARRAY ['active'::character varying, 'inactive'::character varying, 'pending_setup'::character varying, 'suspended'::character varying])::text[])),
    created_at     timestamp   default CURRENT_TIMESTAMP                  not null,
    created_by     bigint                                                 not null,
    updated_at     timestamp   default CURRENT_TIMESTAMP                  not null,
    updated_by     bigint                                                 not null,
    is_deleted     boolean     default false                              not null
);

alter table organizations
    owner to appdb_admin;

grant select, update, usage on sequence organizations_id_seq to app_user;

create index idx_organizations_company_type
    on organizations (org_type);

create index idx_organizations_status
    on organizations (status);

create index idx_organizations_is_deleted
    on organizations (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on organizations to app_user;

create table locations
(
    id            bigserial
        primary key,
    org_id        bigint                                           not null
        constraint fk_locations_org
            references organizations,
    name          varchar(255)                                     not null,
    location_type varchar(50)  default 'office'::character varying not null
        constraint locations_location_type_check
            check ((location_type)::text = ANY
                   ((ARRAY ['office'::character varying, 'warehouse'::character varying, 'job_site'::character varying, 'yard'::character varying])::text[])),
    address       text,
    city          varchar(100),
    state         varchar(50),
    zip_code      varchar(20),
    country       varchar(100) default 'USA'::character varying,
    status        varchar(50)  default 'active'::character varying not null
        constraint locations_status_check
            check ((status)::text = ANY
                   ((ARRAY ['active'::character varying, 'inactive'::character varying, 'under_construction'::character varying, 'closed'::character varying])::text[])),
    created_at    timestamp    default CURRENT_TIMESTAMP           not null,
    created_by    bigint                                           not null,
    updated_at    timestamp    default CURRENT_TIMESTAMP           not null,
    updated_by    bigint                                           not null,
    is_deleted    boolean      default false                       not null
);

alter table locations
    owner to appdb_admin;

grant select, update, usage on sequence locations_id_seq to app_user;

create index idx_locations_org_id
    on locations (org_id);

create index idx_locations_type_status
    on locations (location_type, status);

create index idx_locations_is_deleted
    on locations (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on locations to app_user;

create table users
(
    id                        bigserial
        primary key,
    org_id                    bigint                                           not null
        constraint fk_users_org
            references organizations,
    cognito_id                varchar(255)                                     not null,
    email                     varchar(255)                                     not null,
    first_name                varchar(100),
    last_name                 varchar(100),
    phone                     varchar(20),
    mobile                    varchar(20),
    job_title                 varchar(100),
    employee_id               varchar(50),
    avatar_url                varchar(500),
    last_selected_location_id bigint
        constraint fk_users_last_selected_location
            references locations,
    is_super_admin            boolean     default false                        not null,
    status                    varchar(50) default 'pending'::character varying not null
        constraint users_status_check
            check ((status)::text = ANY
                   ((ARRAY ['active'::character varying, 'inactive'::character varying, 'pending'::character varying, 'pending_org_setup'::character varying, 'suspended'::character varying])::text[])),
    created_at                timestamp   default CURRENT_TIMESTAMP            not null,
    created_by                bigint                                           not null,
    updated_at                timestamp   default CURRENT_TIMESTAMP            not null,
    updated_by                bigint                                           not null,
    is_deleted                boolean     default false                        not null
);

alter table users
    owner to appdb_admin;

grant select, update, usage on sequence users_id_seq to app_user;

create index idx_users_org_id
    on users (org_id);

create index idx_users_cognito_id
    on users (cognito_id);

create index idx_users_email
    on users (email);

create index idx_users_employee_id
    on users (employee_id);

create index idx_users_status
    on users (status);

create index idx_users_is_deleted
    on users (is_deleted);

create index idx_users_last_selected_location
    on users (last_selected_location_id);

grant delete, insert, references, select, trigger, truncate, update on users to app_user;

create table user_location_access
(
    id          bigserial
        primary key,
    user_id     bigint                              not null
        constraint fk_user_location_access_user
            references users,
    location_id bigint                              not null
        constraint fk_user_location_access_location
            references locations,
    is_default  boolean   default false             not null,
    created_at  timestamp default CURRENT_TIMESTAMP not null,
    created_by  bigint                              not null,
    updated_at  timestamp default CURRENT_TIMESTAMP not null,
    updated_by  bigint                              not null,
    is_deleted  boolean   default false             not null
);

alter table user_location_access
    owner to appdb_admin;

grant select, update, usage on sequence user_location_access_id_seq to app_user;

create index idx_user_location_access_user_id
    on user_location_access (user_id);

create index idx_user_location_access_location_id
    on user_location_access (location_id);

create index idx_user_location_access_is_deleted
    on user_location_access (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on user_location_access to app_user;

create table roles
(
    id                         bigserial
        primary key,
    org_id                     bigint
        constraint fk_roles_org
            references organizations,
    name                       varchar(100)                                      not null,
    description                text,
    role_type                  varchar(50) default 'custom'::character varying   not null
        constraint roles_role_type_check
            check ((role_type)::text = ANY
                   ((ARRAY ['system'::character varying, 'custom'::character varying])::text[])),
    construction_role_category varchar(50)                                       not null
        constraint roles_construction_role_category_check
            check ((construction_role_category)::text = ANY
                   ((ARRAY ['management'::character varying, 'field'::character varying, 'office'::character varying, 'external'::character varying, 'admin'::character varying])::text[])),
    access_level               varchar(50) default 'location'::character varying not null
        constraint roles_access_level_check
            check ((access_level)::text = ANY
                   ((ARRAY ['organization'::character varying, 'location'::character varying, 'project'::character varying])::text[])),
    created_at                 timestamp   default CURRENT_TIMESTAMP             not null,
    created_by                 bigint                                            not null,
    updated_at                 timestamp   default CURRENT_TIMESTAMP             not null,
    updated_by                 bigint                                            not null,
    is_deleted                 boolean     default false                         not null
);

alter table roles
    owner to appdb_admin;

grant select, update, usage on sequence roles_id_seq to app_user;

create index idx_roles_org_id
    on roles (org_id);

create index idx_roles_type_category
    on roles (role_type, construction_role_category);

create index idx_roles_access_level
    on roles (access_level);

create index idx_roles_is_deleted
    on roles (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on roles to app_user;

create table permissions
(
    id              bigserial
        primary key,
    code            varchar(100)                                    not null,
    name            varchar(150)                                    not null,
    description     text,
    permission_type varchar(50) default 'system'::character varying not null
        constraint permissions_permission_type_check
            check ((permission_type)::text = ANY
                   ((ARRAY ['system'::character varying, 'custom'::character varying])::text[])),
    module          varchar(50)                                     not null,
    resource_type   varchar(50),
    action_type     varchar(50),
    created_at      timestamp   default CURRENT_TIMESTAMP           not null,
    created_by      bigint                                          not null,
    updated_at      timestamp   default CURRENT_TIMESTAMP           not null,
    updated_by      bigint                                          not null,
    is_deleted      boolean     default false                       not null
);

alter table permissions
    owner to appdb_admin;

grant select, update, usage on sequence permissions_id_seq to app_user;

create index idx_permissions_code
    on permissions (code);

create index idx_permissions_module
    on permissions (module);

create index idx_permissions_resource_action
    on permissions (resource_type, action_type);

create index idx_permissions_type
    on permissions (permission_type);

create index idx_permissions_is_deleted
    on permissions (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on permissions to app_user;

create table role_permissions
(
    role_id       bigint                              not null
        constraint fk_role_permissions_role
            references roles,
    permission_id bigint                              not null
        constraint fk_role_permissions_permission
            references permissions,
    created_at    timestamp default CURRENT_TIMESTAMP not null,
    created_by    bigint                              not null,
    updated_at    timestamp default CURRENT_TIMESTAMP not null,
    updated_by    bigint                              not null,
    is_deleted    boolean   default false             not null,
    primary key (role_id, permission_id)
);

alter table role_permissions
    owner to appdb_admin;

create index idx_role_permissions_is_deleted
    on role_permissions (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on role_permissions to app_user;

create table org_user_roles
(
    id         bigserial
        primary key,
    user_id    bigint                              not null
        constraint fk_org_user_roles_user
            references users,
    role_id    bigint                              not null
        constraint fk_org_user_roles_role
            references roles,
    created_at timestamp default CURRENT_TIMESTAMP not null,
    created_by bigint                              not null,
    updated_at timestamp default CURRENT_TIMESTAMP not null,
    updated_by bigint                              not null,
    is_deleted boolean   default false             not null
);

alter table org_user_roles
    owner to appdb_admin;

grant select, update, usage on sequence org_user_roles_id_seq to app_user;

create index idx_org_user_roles_user_id
    on org_user_roles (user_id);

create index idx_org_user_roles_role_id
    on org_user_roles (role_id);

create index idx_org_user_roles_is_deleted
    on org_user_roles (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on org_user_roles to app_user;

create table location_user_roles
(
    id          bigserial
        primary key,
    user_id     bigint                              not null
        constraint fk_location_user_roles_user
            references users,
    location_id bigint                              not null
        constraint fk_location_user_roles_location
            references locations,
    role_id     bigint                              not null
        constraint fk_location_user_roles_role
            references roles,
    created_at  timestamp default CURRENT_TIMESTAMP not null,
    created_by  bigint                              not null,
    updated_at  timestamp default CURRENT_TIMESTAMP not null,
    updated_by  bigint                              not null,
    is_deleted  boolean   default false             not null
);

alter table location_user_roles
    owner to appdb_admin;

grant select, update, usage on sequence location_user_roles_id_seq to app_user;

create index idx_location_user_roles_user_id
    on location_user_roles (user_id);

create index idx_location_user_roles_location_id
    on location_user_roles (location_id);

create index idx_location_user_roles_role_id
    on location_user_roles (role_id);

create index idx_location_user_roles_is_deleted
    on location_user_roles (is_deleted);

grant delete, insert, references, select, trigger, truncate, update on location_user_roles to app_user;

create function update_updated_at_column() returns trigger
    language plpgsql
as
$$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

alter function update_updated_at_column() owner to appdb_admin;

create trigger update_organizations_updated_at
    before update
    on organizations
    for each row
execute procedure update_updated_at_column();

create trigger update_locations_updated_at
    before update
    on locations
    for each row
execute procedure update_updated_at_column();

create trigger update_users_updated_at
    before update
    on users
    for each row
execute procedure update_updated_at_column();

create trigger update_user_location_access_updated_at
    before update
    on user_location_access
    for each row
execute procedure update_updated_at_column();

create trigger update_roles_updated_at
    before update
    on roles
    for each row
execute procedure update_updated_at_column();

create trigger update_permissions_updated_at
    before update
    on permissions
    for each row
execute procedure update_updated_at_column();

create trigger update_role_permissions_updated_at
    before update
    on role_permissions
    for each row
execute procedure update_updated_at_column();

create trigger update_org_user_roles_updated_at
    before update
    on org_user_roles
    for each row
execute procedure update_updated_at_column();

create trigger update_location_user_roles_updated_at
    before update
    on location_user_roles
    for each row
execute procedure update_updated_at_column();

create table projects
(
    id                          bigserial
        primary key,
    org_id                      bigint                                                     not null
        constraint fk_projects_org
            references iam.organizations,
    location_id                 bigint                                                     not null
        constraint fk_projects_location
            references iam.locations,
    project_number              varchar(50),
    name                        varchar(255)                                               not null,
    description                 text,
    project_type                varchar(50)                                                not null
        constraint projects_project_type_check
            check ((project_type)::text = ANY
                   ((ARRAY ['commercial'::character varying, 'residential'::character varying, 'industrial'::character varying, 'hospitality'::character varying, 'healthcare'::character varying, 'institutional'::character varying, 'mixed-use'::character varying, 'civil-infrastructure'::character varying, 'recreation'::character varying, 'aviation'::character varying, 'specialized'::character varying])::text[])),
    project_stage               varchar(50)
        constraint projects_project_stage_check
            check ((project_stage)::text = ANY
                   ((ARRAY ['bidding'::character varying, 'course-of-construction'::character varying, 'pre-construction'::character varying, 'post-construction'::character varying, 'warranty'::character varying])::text[])),
    work_scope                  varchar(50)
        constraint projects_work_scope_check
            check ((work_scope)::text = ANY
                   ((ARRAY ['new'::character varying, 'renovation'::character varying, 'restoration'::character varying, 'maintenance'::character varying])::text[])),
    project_sector              varchar(50)
        constraint projects_project_sector_check
            check ((project_sector)::text = ANY
                   ((ARRAY ['commercial'::character varying, 'residential'::character varying, 'industrial'::character varying, 'hospitality'::character varying, 'healthcare'::character varying, 'institutional'::character varying, 'mixed-use'::character varying, 'civil-infrastructure'::character varying, 'recreation'::character varying, 'aviation'::character varying, 'specialized'::character varying])::text[])),
    delivery_method             varchar(50)
        constraint projects_delivery_method_check
            check ((delivery_method)::text = ANY
                   ((ARRAY ['design-build'::character varying, 'design-bid-build'::character varying, 'construction-manager-at-risk'::character varying, 'integrated-project-delivery'::character varying, 'construction-manager-as-agent'::character varying, 'public-private-partnership'::character varying, 'other'::character varying])::text[])),
    project_phase               varchar(50)  default 'pre_construction'::character varying not null
        constraint projects_project_phase_check
            check ((project_phase)::text = ANY
                   ((ARRAY ['pre_construction'::character varying, 'design'::character varying, 'permitting'::character varying, 'construction'::character varying, 'closeout'::character varying, 'warranty'::character varying])::text[])),
    start_date                  date,
    planned_end_date            date,
    actual_start_date           date,
    actual_end_date             date,
    substantial_completion_date date,
    project_finish_date         date,
    warranty_start_date         date,
    warranty_end_date           date,
    budget                      numeric(15, 2),
    contract_value              numeric(15, 2),
    square_footage              integer,
    address                     text,
    city                        varchar(100),
    state                       varchar(50),
    zip_code                    varchar(20),
    country                     varchar(100) default 'USA'::character varying,
    language                    varchar(10)  default 'en'::character varying,
    latitude                    numeric(10, 8),
    longitude                   numeric(11, 8),
    status                      varchar(50)  default 'active'::character varying           not null
        constraint projects_status_check
            check ((status)::text = ANY
                   ((ARRAY ['active'::character varying, 'inactive'::character varying, 'on_hold'::character varying, 'completed'::character varying, 'cancelled'::character varying])::text[])),
    created_at                  timestamp    default CURRENT_TIMESTAMP                     not null,
    created_by                  bigint                                                     not null,
    updated_at                  timestamp    default CURRENT_TIMESTAMP                     not null,
    updated_by                  bigint                                                     not null,
    is_deleted                  boolean      default false                                 not null
);

alter table projects
    owner to appdb_admin;

grant select, update, usage on sequence projects_id_seq to app_user;

create index idx_projects_org_id
    on projects (org_id);

create index idx_projects_location_id
    on projects (location_id);

create index idx_projects_number
    on projects (project_number);

create index idx_projects_type_phase
    on projects (project_type, project_phase);

create index idx_projects_stage
    on projects (project_stage);

create index idx_projects_sector
    on projects (project_sector);

create index idx_projects_delivery_method
    on projects (delivery_method);

create index idx_projects_dates
    on projects (start_date, planned_end_date);

create index idx_projects_status
    on projects (status);

create index idx_projects_is_deleted
    on projects (is_deleted);

create trigger update_projects_updated_at
    before update
    on projects
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on projects to app_user;

create table project_managers
(
    id             bigserial
        primary key,
    project_id     bigint                              not null
        constraint fk_project_managers_project
            references projects,
    name           varchar(255)                        not null,
    company        varchar(255)                        not null,
    role           varchar(100)                        not null
        constraint project_managers_role_check
            check ((role)::text = ANY
                   ((ARRAY ['general-contractor'::character varying, 'owners-representative'::character varying, 'program-manager'::character varying, 'consultant'::character varying, 'architect'::character varying, 'engineer'::character varying, 'inspector'::character varying])::text[])),
    email          varchar(255)                        not null,
    office_contact varchar(20),
    mobile_contact varchar(20),
    is_primary     boolean   default true,
    created_at     timestamp default CURRENT_TIMESTAMP not null,
    created_by     bigint                              not null,
    updated_at     timestamp default CURRENT_TIMESTAMP not null,
    updated_by     bigint                              not null,
    is_deleted     boolean   default false             not null
);

alter table project_managers
    owner to appdb_admin;

grant select, update, usage on sequence project_managers_id_seq to app_user;

create index idx_project_managers_project_id
    on project_managers (project_id);

create index idx_project_managers_email
    on project_managers (email);

create index idx_project_managers_role
    on project_managers (role);

create index idx_project_managers_is_deleted
    on project_managers (is_deleted);

create trigger update_project_managers_updated_at
    before update
    on project_managers
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on project_managers to app_user;

create table project_attachments
(
    id              bigserial
        primary key,
    project_id      bigint                              not null
        constraint fk_project_attachments_project
            references projects,
    file_name       varchar(255)                        not null,
    file_path       varchar(500)                        not null,
    file_size       bigint,
    file_type       varchar(50),
    attachment_type varchar(50)                         not null
        constraint project_attachments_attachment_type_check
            check ((attachment_type)::text = ANY
                   ((ARRAY ['logo'::character varying, 'project_photo'::character varying, 'document'::character varying, 'drawing'::character varying, 'other'::character varying])::text[])),
    uploaded_by     bigint                              not null
        constraint fk_project_attachments_uploaded_by
            references iam.users,
    created_at      timestamp default CURRENT_TIMESTAMP not null,
    created_by      bigint                              not null,
    updated_at      timestamp default CURRENT_TIMESTAMP not null,
    updated_by      bigint                              not null,
    is_deleted      boolean   default false             not null
);

alter table project_attachments
    owner to appdb_admin;

grant select, update, usage on sequence project_attachments_id_seq to app_user;

create index idx_project_attachments_project_id
    on project_attachments (project_id);

create index idx_project_attachments_type
    on project_attachments (attachment_type);

create index idx_project_attachments_is_deleted
    on project_attachments (is_deleted);

create trigger update_project_attachments_updated_at
    before update
    on project_attachments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on project_attachments to app_user;

create table project_user_roles
(
    id         bigserial
        primary key,
    project_id bigint                              not null
        constraint fk_project_user_roles_project
            references projects,
    user_id    bigint                              not null
        constraint fk_project_user_roles_user
            references iam.users,
    role_id    bigint                              not null
        constraint fk_project_user_roles_role
            references iam.roles,
    trade_type varchar(100),
    is_primary boolean   default false             not null,
    start_date date,
    end_date   date,
    created_at timestamp default CURRENT_TIMESTAMP not null,
    created_by bigint                              not null,
    updated_at timestamp default CURRENT_TIMESTAMP not null,
    updated_by bigint                              not null,
    is_deleted boolean   default false             not null
);

alter table project_user_roles
    owner to appdb_admin;

grant select, update, usage on sequence project_user_roles_id_seq to app_user;

create index idx_project_user_roles_project_id
    on project_user_roles (project_id);

create index idx_project_user_roles_user_id
    on project_user_roles (user_id);

create index idx_project_user_roles_role_id
    on project_user_roles (role_id);

create index idx_project_user_roles_trade
    on project_user_roles (trade_type);

create index idx_project_user_roles_primary
    on project_user_roles (is_primary);

create index idx_project_user_roles_is_deleted
    on project_user_roles (is_deleted);

create trigger update_project_user_roles_updated_at
    before update
    on project_user_roles
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on project_user_roles to app_user;

create table rfis
(
    id                      bigserial
        primary key,
    project_id              bigint                                             not null
        constraint fk_rfis_project
            references projects,
    rfi_number              varchar(50)                                        not null,
    title                   varchar(255)                                       not null,
    description             text                                               not null,
    question                text                                               not null,
    location_description    varchar(255),
    drawing_reference       varchar(255),
    specification_reference varchar(255),
    priority                varchar(50)    default 'medium'::character varying not null
        constraint rfis_priority_check
            check ((priority)::text = ANY
                   ((ARRAY ['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[])),
    status                  varchar(50)    default 'draft'::character varying  not null
        constraint rfis_status_check
            check ((status)::text = ANY
                   ((ARRAY ['draft'::character varying, 'submitted'::character varying, 'in_review'::character varying, 'responded'::character varying, 'closed'::character varying, 'cancelled'::character varying])::text[])),
    submitted_by            bigint                                             not null
        constraint fk_rfis_submitted_by
            references iam.users,
    assigned_to             bigint
        constraint fk_rfis_assigned_to
            references iam.users,
    submitted_date          timestamp,
    due_date                timestamp,
    response_date           timestamp,
    response                text,
    response_by             bigint
        constraint fk_rfis_response_by
            references iam.users,
    cost_impact             numeric(15, 2) default 0.00,
    schedule_impact_days    integer        default 0,
    trade_type              varchar(100),
    created_at              timestamp      default CURRENT_TIMESTAMP           not null,
    created_by              bigint                                             not null,
    updated_at              timestamp      default CURRENT_TIMESTAMP           not null,
    updated_by              bigint                                             not null,
    is_deleted              boolean        default false                       not null
);

alter table rfis
    owner to appdb_admin;

grant select, update, usage on sequence rfis_id_seq to app_user;

create index idx_rfis_project_id
    on rfis (project_id);

create index idx_rfis_number
    on rfis (rfi_number);

create index idx_rfis_status
    on rfis (status);

create index idx_rfis_priority
    on rfis (priority);

create index idx_rfis_submitted_by
    on rfis (submitted_by);

create index idx_rfis_assigned_to
    on rfis (assigned_to);

create index idx_rfis_due_date
    on rfis (due_date);

create index idx_rfis_trade_type
    on rfis (trade_type);

create index idx_rfis_is_deleted
    on rfis (is_deleted);

create trigger update_rfis_updated_at
    before update
    on rfis
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on rfis to app_user;

create table rfi_attachments
(
    id          bigserial
        primary key,
    rfi_id      bigint                              not null
        constraint fk_rfi_attachments_rfi
            references rfis,
    file_name   varchar(255)                        not null,
    file_path   varchar(500)                        not null,
    file_size   bigint,
    file_type   varchar(50),
    uploaded_by bigint                              not null
        constraint fk_rfi_attachments_uploaded_by
            references iam.users,
    created_at  timestamp default CURRENT_TIMESTAMP not null,
    created_by  bigint                              not null,
    updated_at  timestamp default CURRENT_TIMESTAMP not null,
    updated_by  bigint                              not null,
    is_deleted  boolean   default false             not null
);

alter table rfi_attachments
    owner to appdb_admin;

grant select, update, usage on sequence rfi_attachments_id_seq to app_user;

create index idx_rfi_attachments_rfi_id
    on rfi_attachments (rfi_id);

create index idx_rfi_attachments_is_deleted
    on rfi_attachments (is_deleted);

create trigger update_rfi_attachments_updated_at
    before update
    on rfi_attachments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on rfi_attachments to app_user;

create table rfi_comments
(
    id             bigserial
        primary key,
    rfi_id         bigint                                           not null
        constraint fk_rfi_comments_rfi
            references rfis,
    comment        text                                             not null,
    comment_type   varchar(50) default 'comment'::character varying not null
        constraint rfi_comments_comment_type_check
            check ((comment_type)::text = ANY
                   ((ARRAY ['comment'::character varying, 'status_change'::character varying, 'assignment'::character varying, 'response'::character varying])::text[])),
    previous_value varchar(255),
    new_value      varchar(255),
    created_at     timestamp   default CURRENT_TIMESTAMP            not null,
    created_by     bigint                                           not null
        constraint fk_rfi_comments_created_by
            references iam.users,
    updated_at     timestamp   default CURRENT_TIMESTAMP            not null,
    updated_by     bigint                                           not null,
    is_deleted     boolean     default false                        not null
);

alter table rfi_comments
    owner to appdb_admin;

grant select, update, usage on sequence rfi_comments_id_seq to app_user;

create index idx_rfi_comments_rfi_id
    on rfi_comments (rfi_id);

create index idx_rfi_comments_type
    on rfi_comments (comment_type);

create index idx_rfi_comments_is_deleted
    on rfi_comments (is_deleted);

create trigger update_rfi_comments_updated_at
    before update
    on rfi_comments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on rfi_comments to app_user;

create table submittals
(
    id                    bigserial
        primary key,
    project_id            bigint                                          not null
        constraint fk_submittals_project
            references projects,
    submittal_number      varchar(50)                                     not null,
    title                 varchar(255)                                    not null,
    description           text,
    submittal_type        varchar(50)                                     not null
        constraint submittals_submittal_type_check
            check ((submittal_type)::text = ANY
                   ((ARRAY ['shop_drawings'::character varying, 'product_data'::character varying, 'samples'::character varying, 'design_mix'::character varying, 'test_reports'::character varying, 'certificates'::character varying, 'operation_manuals'::character varying, 'warranties'::character varying])::text[])),
    specification_section varchar(50),
    drawing_reference     varchar(255),
    trade_type            varchar(100),
    priority              varchar(50) default 'medium'::character varying not null
        constraint submittals_priority_check
            check ((priority)::text = ANY
                   ((ARRAY ['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[])),
    status                varchar(50) default 'draft'::character varying  not null
        constraint submittals_status_check
            check ((status)::text = ANY
                   ((ARRAY ['draft'::character varying, 'submitted'::character varying, 'under_review'::character varying, 'approved'::character varying, 'approved_with_comments'::character varying, 'rejected'::character varying, 'resubmit_required'::character varying])::text[])),
    revision_number       integer     default 1                           not null,
    submitted_by          bigint                                          not null
        constraint fk_submittals_submitted_by
            references iam.users,
    submitted_company_id  bigint
        constraint fk_submittals_submitted_company
            references iam.organizations,
    reviewed_by           bigint
        constraint fk_submittals_reviewed_by
            references iam.users,
    submitted_date        timestamp,
    due_date              timestamp,
    reviewed_date         timestamp,
    approval_date         timestamp,
    review_comments       text,
    lead_time_days        integer,
    quantity_submitted    integer,
    unit_of_measure       varchar(20),
    created_at            timestamp   default CURRENT_TIMESTAMP           not null,
    created_by            bigint                                          not null,
    updated_at            timestamp   default CURRENT_TIMESTAMP           not null,
    updated_by            bigint                                          not null,
    is_deleted            boolean     default false                       not null
);

alter table submittals
    owner to appdb_admin;

grant select, update, usage on sequence submittals_id_seq to app_user;

create index idx_submittals_project_id
    on submittals (project_id);

create index idx_submittals_number
    on submittals (submittal_number);

create index idx_submittals_type
    on submittals (submittal_type);

create index idx_submittals_status
    on submittals (status);

create index idx_submittals_priority
    on submittals (priority);

create index idx_submittals_submitted_by
    on submittals (submitted_by);

create index idx_submittals_reviewed_by
    on submittals (reviewed_by);

create index idx_submittals_spec_section
    on submittals (specification_section);

create index idx_submittals_trade_type
    on submittals (trade_type);

create index idx_submittals_due_date
    on submittals (due_date);

create index idx_submittals_is_deleted
    on submittals (is_deleted);

create trigger update_submittals_updated_at
    before update
    on submittals
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on submittals to app_user;

create table submittal_items
(
    id               bigserial
        primary key,
    submittal_id     bigint                                           not null
        constraint fk_submittal_items_submittal
            references submittals,
    item_number      varchar(50),
    item_description text                                             not null,
    manufacturer     varchar(255),
    model_number     varchar(100),
    quantity         integer,
    unit_price       numeric(15, 2),
    total_price      numeric(15, 2),
    status           varchar(50) default 'pending'::character varying not null
        constraint submittal_items_status_check
            check ((status)::text = ANY
                   ((ARRAY ['pending'::character varying, 'approved'::character varying, 'rejected'::character varying, 'approved_with_comments'::character varying])::text[])),
    comments         text,
    created_at       timestamp   default CURRENT_TIMESTAMP            not null,
    created_by       bigint                                           not null,
    updated_at       timestamp   default CURRENT_TIMESTAMP            not null,
    updated_by       bigint                                           not null,
    is_deleted       boolean     default false                        not null
);

alter table submittal_items
    owner to appdb_admin;

grant select, update, usage on sequence submittal_items_id_seq to app_user;

create index idx_submittal_items_submittal_id
    on submittal_items (submittal_id);

create index idx_submittal_items_status
    on submittal_items (status);

create index idx_submittal_items_is_deleted
    on submittal_items (is_deleted);

create trigger update_submittal_items_updated_at
    before update
    on submittal_items
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on submittal_items to app_user;

create table submittal_attachments
(
    id              bigserial
        primary key,
    submittal_id    bigint                                         not null
        constraint fk_submittal_attachments_submittal
            references submittals,
    file_name       varchar(255)                                   not null,
    file_path       varchar(500)                                   not null,
    file_size       bigint,
    file_type       varchar(50),
    attachment_type varchar(50) default 'other'::character varying not null
        constraint submittal_attachments_attachment_type_check
            check ((attachment_type)::text = ANY
                   ((ARRAY ['shop_drawing'::character varying, 'product_data'::character varying, 'specification'::character varying, 'sample_photo'::character varying, 'certificate'::character varying, 'test_report'::character varying, 'other'::character varying])::text[])),
    uploaded_by     bigint                                         not null
        constraint fk_submittal_attachments_uploaded_by
            references iam.users,
    created_at      timestamp   default CURRENT_TIMESTAMP          not null,
    created_by      bigint                                         not null,
    updated_at      timestamp   default CURRENT_TIMESTAMP          not null,
    updated_by      bigint                                         not null,
    is_deleted      boolean     default false                      not null
);

alter table submittal_attachments
    owner to appdb_admin;

grant select, update, usage on sequence submittal_attachments_id_seq to app_user;

create index idx_submittal_attachments_submittal_id
    on submittal_attachments (submittal_id);

create index idx_submittal_attachments_type
    on submittal_attachments (attachment_type);

create index idx_submittal_attachments_is_deleted
    on submittal_attachments (is_deleted);

create trigger update_submittal_attachments_updated_at
    before update
    on submittal_attachments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on submittal_attachments to app_user;

create table submittal_reviews
(
    id              bigserial
        primary key,
    submittal_id    bigint                              not null
        constraint fk_submittal_reviews_submittal
            references submittals,
    revision_number integer                             not null,
    reviewer_id     bigint                              not null
        constraint fk_submittal_reviews_reviewer
            references iam.users,
    review_status   varchar(50)                         not null
        constraint submittal_reviews_review_status_check
            check ((review_status)::text = ANY
                   ((ARRAY ['approved'::character varying, 'approved_with_comments'::character varying, 'rejected'::character varying, 'resubmit_required'::character varying])::text[])),
    review_comments text,
    review_date     timestamp default CURRENT_TIMESTAMP not null,
    created_at      timestamp default CURRENT_TIMESTAMP not null,
    created_by      bigint                              not null,
    updated_at      timestamp default CURRENT_TIMESTAMP not null,
    updated_by      bigint                              not null,
    is_deleted      boolean   default false             not null
);

alter table submittal_reviews
    owner to appdb_admin;

grant select, update, usage on sequence submittal_reviews_id_seq to app_user;

create index idx_submittal_reviews_submittal_id
    on submittal_reviews (submittal_id);

create index idx_submittal_reviews_reviewer_id
    on submittal_reviews (reviewer_id);

create index idx_submittal_reviews_status
    on submittal_reviews (review_status);

create index idx_submittal_reviews_is_deleted
    on submittal_reviews (is_deleted);

create trigger update_submittal_reviews_updated_at
    before update
    on submittal_reviews
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on submittal_reviews to app_user;

create table issue_templates
(
    id                  bigserial
        primary key,
    org_id              bigint                              not null
        constraint fk_issue_templates_org
            references iam.organizations,
    name                varchar(255)                        not null,
    category            varchar(100),
    detail_category     varchar(100),
    default_priority    varchar(50),
    default_severity    varchar(50),
    default_description text,
    is_active           boolean   default true,
    created_at          timestamp default CURRENT_TIMESTAMP not null,
    created_by          bigint                              not null,
    updated_at          timestamp default CURRENT_TIMESTAMP not null,
    updated_by          bigint                              not null
);

alter table issue_templates
    owner to appdb_admin;

create table issues
(
    id                      bigserial
        primary key,
    project_id              bigint                                              not null
        constraint fk_issues_project
            references projects,
    issue_number            varchar(50)                                         not null,
    title                   varchar(255)                                        not null,
    description             text                                                not null,
    issue_type              varchar(50)    default 'general'::character varying not null
        constraint issues_issue_type_check
            check ((issue_type)::text = ANY
                   ((ARRAY ['quality'::character varying, 'safety'::character varying, 'deficiency'::character varying, 'punch_item'::character varying, 'code_violation'::character varying, 'general'::character varying])::text[])),
    severity                varchar(50)    default 'minor'::character varying   not null
        constraint issues_severity_check
            check ((severity)::text = ANY
                   ((ARRAY ['blocking'::character varying, 'major'::character varying, 'minor'::character varying, 'cosmetic'::character varying])::text[])),
    priority                varchar(50)    default 'medium'::character varying  not null
        constraint issues_priority_check
            check ((priority)::text = ANY
                   ((ARRAY ['critical'::character varying, 'high'::character varying, 'medium'::character varying, 'low'::character varying, 'planned'::character varying])::text[])),
    status                  varchar(50)    default 'open'::character varying    not null
        constraint issues_status_check
            check ((status)::text = ANY
                   ((ARRAY ['open'::character varying, 'in_progress'::character varying, 'ready_for_review'::character varying, 'closed'::character varying, 'rejected'::character varying, 'on_hold'::character varying])::text[])),
    location_description    varchar(255),
    room_area               varchar(100),
    floor_level             varchar(50),
    trade_type              varchar(100),
    reported_by             bigint                                              not null
        constraint fk_issues_reported_by
            references iam.users,
    assigned_to             bigint
        constraint fk_issues_assigned_to
            references iam.users,
    assigned_company_id     bigint
        constraint fk_issues_assigned_company
            references iam.organizations,
    due_date                date,
    closed_date             timestamp,
    cost_to_fix             numeric(15, 2) default 0.00,
    drawing_reference       varchar(255),
    specification_reference varchar(255),
    latitude                numeric(10, 8),
    longitude               numeric(11, 8),
    created_at              timestamp      default CURRENT_TIMESTAMP            not null,
    created_by              bigint                                              not null,
    updated_at              timestamp      default CURRENT_TIMESTAMP            not null,
    updated_by              bigint                                              not null,
    is_deleted              boolean        default false                        not null,
    template_id             bigint
        constraint fk_issues_template
            references issue_templates,
    category                varchar(100),
    detail_category         varchar(100),
    root_cause              text,
    discipline              varchar(100),
    location_building       varchar(100),
    location_level          varchar(50),
    location_room           varchar(100),
    location_x              numeric(10, 4),
    location_y              numeric(10, 4),
    distribution_list       text[],
    issue_category          varchar(100)
        constraint issues_issue_category_check
            check ((issue_category IS NULL) OR ((issue_category)::text = ANY
                                                (ARRAY [('quality'::character varying)::text, ('safety'::character varying)::text, ('deficiency'::character varying)::text, ('punch_item'::character varying)::text, ('code_violation'::character varying)::text, ('general'::character varying)::text])))
);

alter table issues
    owner to appdb_admin;

grant select, update, usage on sequence issues_id_seq to app_user;

create index idx_issues_project_id
    on issues (project_id);

create index idx_issues_number
    on issues (issue_number);

create index idx_issues_type_severity
    on issues (issue_type, severity);

create index idx_issues_status
    on issues (status);

create index idx_issues_priority
    on issues (priority);

create index idx_issues_reported_by
    on issues (reported_by);

create index idx_issues_assigned_to
    on issues (assigned_to);

create index idx_issues_trade_type
    on issues (trade_type);

create index idx_issues_due_date
    on issues (due_date);

create index idx_issues_is_deleted
    on issues (is_deleted);

create index idx_issues_category
    on issues (category);

create index idx_issues_template
    on issues (template_id);

create index idx_issues_discipline
    on issues (discipline);

create index idx_issues_issue_category
    on issues (issue_category);

create trigger update_issues_updated_at
    before update
    on issues
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on issues to app_user;

create table issue_attachments
(
    id              bigserial
        primary key,
    issue_id        bigint                                                not null
        constraint fk_issue_attachments_issue
            references issues,
    file_name       varchar(255)                                          not null,
    file_path       varchar(500)                                          not null,
    file_size       bigint,
    file_type       varchar(50),
    attachment_type varchar(50) default 'before_photo'::character varying not null
        constraint issue_attachments_attachment_type_check
            check ((attachment_type)::text = ANY
                   ((ARRAY ['before_photo'::character varying, 'after_photo'::character varying, 'document'::character varying, 'drawing_markup'::character varying])::text[])),
    uploaded_by     bigint                                                not null
        constraint fk_issue_attachments_uploaded_by
            references iam.users,
    created_at      timestamp   default CURRENT_TIMESTAMP                 not null,
    created_by      bigint                                                not null,
    updated_at      timestamp   default CURRENT_TIMESTAMP                 not null,
    updated_by      bigint                                                not null,
    is_deleted      boolean     default false                             not null
);

alter table issue_attachments
    owner to appdb_admin;

grant select, update, usage on sequence issue_attachments_id_seq to app_user;

create index idx_issue_attachments_issue_id
    on issue_attachments (issue_id);

create index idx_issue_attachments_type
    on issue_attachments (attachment_type);

create index idx_issue_attachments_is_deleted
    on issue_attachments (is_deleted);

create trigger update_issue_attachments_updated_at
    before update
    on issue_attachments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on issue_attachments to app_user;

create table issue_comments
(
    id             bigserial
        primary key,
    issue_id       bigint                                           not null
        constraint fk_issue_comments_issue
            references issues,
    comment        text                                             not null,
    comment_type   varchar(50) default 'comment'::character varying not null
        constraint issue_comments_comment_type_check
            check ((comment_type)::text = ANY
                   ((ARRAY ['comment'::character varying, 'status_change'::character varying, 'assignment'::character varying, 'resolution'::character varying])::text[])),
    previous_value varchar(255),
    new_value      varchar(255),
    created_at     timestamp   default CURRENT_TIMESTAMP            not null,
    created_by     bigint                                           not null
        constraint fk_issue_comments_created_by
            references iam.users,
    updated_at     timestamp   default CURRENT_TIMESTAMP            not null,
    updated_by     bigint                                           not null,
    is_deleted     boolean     default false                        not null
);

alter table issue_comments
    owner to appdb_admin;

grant select, update, usage on sequence issue_comments_id_seq to app_user;

create index idx_issue_comments_issue_id
    on issue_comments (issue_id);

create index idx_issue_comments_type
    on issue_comments (comment_type);

create index idx_issue_comments_is_deleted
    on issue_comments (is_deleted);

create trigger update_issue_comments_updated_at
    before update
    on issue_comments
    for each row
execute procedure iam.update_updated_at_column();

grant delete, insert, references, select, trigger, truncate, update on issue_comments to app_user;