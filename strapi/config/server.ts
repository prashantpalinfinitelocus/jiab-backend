export default ({ env }) => ({
  host: env("HOST", "0.0.0.0"),
  port: env.int("PORT", 1337),
  url: "/strapi", // Prefix all backend URLs with /strapi
  app: {
    keys: env.array("APP_KEYS"),
  },
  admin: {
    url: "/strapi/admin", // Prefix all admin URLs and assets with /strapi
  },
  // prefix: "/api/v1/cms",
});
