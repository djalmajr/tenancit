// Seed/mock data do painel Konvario. SEM imports — apenas dados exportados.

export const DEFINITIONS = [
  {
    id: "def-postgres",
    key: "postgres",
    name: "PostgreSQL Connection",
    status: "active",
    icon: "lucide:database",
    description: "Conexão de banco de dados PostgreSQL dedicada por tenant.",
    fields: [
      { id: "f1", key: "host", label: "Host", data_type: "string", required: true, is_secret: false },
      { id: "f2", key: "port", label: "Porta", data_type: "int", required: true, is_secret: false },
      { id: "f3", key: "database", label: "Database", data_type: "string", required: true, is_secret: false },
      { id: "f4", key: "username", label: "Usuário", data_type: "string", required: true, is_secret: false },
      { id: "f5", key: "password", label: "Senha", data_type: "string", required: true, is_secret: true },
      { id: "f6", key: "sslmode", label: "SSL Mode", data_type: "string", required: false, is_secret: false },
    ],
  },
  {
    id: "def-minio",
    key: "minio",
    name: "MinIO / S3 Bucket",
    status: "active",
    icon: "lucide:hard-drive",
    description: "Configuração de object storage (bucket + credenciais) por tenant.",
    fields: [
      { id: "m1", key: "endpoint", label: "Endpoint", data_type: "string", required: true, is_secret: false },
      { id: "m2", key: "bucket", label: "Bucket", data_type: "string", required: true, is_secret: false },
      { id: "m3", key: "access_key", label: "Access Key", data_type: "string", required: true, is_secret: false },
      { id: "m4", key: "secret_key", label: "Secret Key", data_type: "string", required: true, is_secret: true },
    ],
  },
  {
    id: "def-smtp",
    key: "smtp",
    name: "SMTP",
    status: "inactive",
    icon: "lucide:mail",
    description: "Servidor de e-mail por tenant.",
    fields: [
      { id: "s1", key: "host", label: "Host", data_type: "string", required: true, is_secret: false },
      { id: "s2", key: "username", label: "Usuário", data_type: "string", required: true, is_secret: false },
      { id: "s3", key: "password", label: "Senha", data_type: "string", required: true, is_secret: true },
    ],
  },
];

export const TENANTS = [
  {
    id: "t-acme",
    slug: "acme",
    name: "Acme Corp",
    status: "active",
    domains: [
      { id: "d1", hostname: "app.acme.test" },
      { id: "d2", hostname: "portal.acme.test" },
    ],
    resources: [
      {
        id: "r1",
        definitionKey: "postgres",
        definitionName: "PostgreSQL Connection",
        status: "active",
        values: {
          host: "db.acme.example",
          port: "5432",
          database: "acme_prod",
          username: "acme_app",
          password: null,
          sslmode: "require",
        },
      },
      {
        id: "r2",
        definitionKey: "minio",
        definitionName: "MinIO / S3 Bucket",
        status: "active",
        values: {
          endpoint: "https://s3.example.test",
          bucket: "demo-acme-assets",
          access_key: null,
          secret_key: null,
        },
      },
    ],
  },
  {
    id: "t-globex",
    slug: "globex",
    name: "Globex Ltda",
    status: "active",
    domains: [{ id: "d3", hostname: "portal.globex.test" }],
    resources: [
      {
        id: "r3",
        definitionKey: "postgres",
        definitionName: "PostgreSQL Connection",
        status: "active",
        values: {
          host: "db.globex.example",
          port: "5432",
          database: "globex_prod",
          username: "globex_app",
          password: null,
          sslmode: "require",
        },
      },
    ],
  },
  {
    id: "t-initech",
    slug: "initech",
    name: "Initech",
    status: "inactive",
    domains: [{ id: "d4", hostname: "app.initech.test" }],
    resources: [],
  },
];

export const API_CLIENTS = [
  { id: "c1", name: "consumer-api", status: "active", key_preview: "rt_demo_••••••8f2a", created_at: "2026-04-12" },
  { id: "c2", name: "billing-service", status: "active", key_preview: "rt_demo_••••••1c90", created_at: "2026-05-02" },
  { id: "c3", name: "demo-importer", status: "revoked", key_preview: "rt_demo_••••••44de", created_at: "2026-01-20" },
];
