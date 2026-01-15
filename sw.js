const CACHE_NAME = 'gousto-v2';
const ASSETS_TO_CACHE = [
    './',
    './index.html',
    './style.css',
    './manifest.json'
];

// Install: Cache core assets immediately
self.addEventListener('install', (event) => {
    self.skipWaiting(); // Force this new SW to become active immediately
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(ASSETS_TO_CACHE);
        })
    );
});

// Activate: Clean up old caches
self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keyList) => {
            return Promise.all(keyList.map((key) => {
                if (key !== CACHE_NAME) {
                    return caches.delete(key);
                }
            }));
        }).then(() => self.clients.claim()) // Take control of all clients immediately
    );
});

// Fetch: Network First for data.json, Cache First for images/static
self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);

    // Strategy 1: Network First (Freshness) for data.json
    // We always want the latest recipe list.
    if (url.pathname.endsWith('data.json')) {
        event.respondWith(
            fetch(event.request)
                .then((response) => {
                    return caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, response.clone());
                        return response;
                    });
                })
                .catch(() => caches.match(event.request)) // Fallback to cache if offline
        );
        return;
    }

    // Strategy 2: Stale-While-Revalidate for HTML/CSS/JS
    // Serve from cache fast, but update in background for next time.
    if (url.pathname.endsWith('.html') || url.pathname.endsWith('.css') || url.pathname.endsWith('.js')) {
        event.respondWith(
            caches.match(event.request).then((cachedResponse) => {
                const fetchPromise = fetch(event.request).then((networkResponse) => {
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, networkResponse.clone());
                    });
                    return networkResponse;
                });
                return cachedResponse || fetchPromise;
            })
        );
        return;
    }

    // Strategy 3: Cache First (Performance) for Images
    // Recipe images don't change often.
    if (url.pathname.includes('/images/')) {
        event.respondWith(
            caches.match(event.request).then((response) => {
                return response || fetch(event.request).then((networkResponse) => {
                    return caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, networkResponse.clone());
                        return networkResponse;
                    });
                });
            })
        );
        return;
    }

    // Default: Cache First for everything else
    event.respondWith(
        caches.match(event.request).then((response) => {
            return response || fetch(event.request);
        })
    );
});