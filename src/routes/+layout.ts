// Every route ships as a static HTML shell. Authenticated data is loaded in the
// browser after hydration because the session cookie is HttpOnly.
export const prerender = true;
export const trailingSlash = 'always';
