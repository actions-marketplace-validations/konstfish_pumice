// pumice.js — htmx navigation + link previews

(function () {
    // ── Link preview ──────────────────────────────────────────────
    var previewEl = null;
    var previewCache = {};
    var hoverTimeout = null;
    var currentLink = null;
    var fetchGeneration = 0; // increments on every hide, stales in-flight fetches

    function ensurePreviewEl() {
        if (!previewEl) {
            previewEl = document.createElement('div');
            previewEl.className = 'link-preview';
            document.body.appendChild(previewEl);
        }
        return previewEl;
    }

    var skeletonHTML = '<div class="skeleton-line skeleton-title"></div>' +
        '<div class="skeleton-line skeleton-text"></div>' +
        '<div class="skeleton-line skeleton-text"></div>' +
        '<div class="skeleton-line skeleton-text"></div>';

    function showSkeleton(link) {
        var el = ensurePreviewEl();
        el.innerHTML = skeletonHTML;
        el.classList.add('visible', 'loading');
        positionPreview(link, el);
    }

    function showPreview(link, html) {
        var el = ensurePreviewEl();
        el.innerHTML = html;
        el.classList.remove('loading');
        el.classList.add('visible');
        positionPreview(link, el);
    }

    function positionPreview(link, el) {
        var rect = link.getBoundingClientRect();
        var scrollY = window.scrollY;
        var scrollX = window.scrollX;

        var top = rect.bottom + scrollY + 8;
        var left = rect.left + scrollX;

        var elWidth = 350;
        if (left + elWidth > window.innerWidth + scrollX - 16) {
            left = window.innerWidth + scrollX - elWidth - 16;
        }
        if (left < scrollX + 16) {
            left = scrollX + 16;
        }

        if (top + 250 > window.innerHeight + scrollY) {
            top = rect.top + scrollY - 258;
        }

        el.style.top = top + 'px';
        el.style.left = left + 'px';
    }

    function hidePreview() {
        clearTimeout(hoverTimeout);
        hoverTimeout = null;
        fetchGeneration++;
        currentLink = null;
        if (previewEl) {
            previewEl.classList.remove('visible', 'loading');
        }
    }

    function fetchPreview(link) {
        var href = link.getAttribute('href');
        var fetchUrl = href;
        if (!fetchUrl.endsWith('.html')) {
            fetchUrl = fetchUrl + '.html';
        }

        if (previewCache[fetchUrl]) {
            if (currentLink === link) {
                showPreview(link, previewCache[fetchUrl]);
            }
            return;
        }

        // Show skeleton immediately while fetching
        showSkeleton(link);

        var gen = fetchGeneration;
        fetch(fetchUrl)
            .then(function (r) { return r.ok ? r.text() : null; })
            .then(function (html) {
                if (!html) return;
                // Stale — a hide happened since this fetch started
                if (gen !== fetchGeneration) return;

                var parser = new DOMParser();
                var doc = parser.parseFromString(html, 'text/html');
                var content = doc.querySelector('#content');
                if (!content) return;

                // Strip tags, meta, backlinks, and giscus from preview
                var strip = content.querySelectorAll('.page-tags, .page-meta, .backlinks, .giscus-comments');
                strip.forEach(function (el) { el.remove(); });

                var preview = content.innerHTML;
                previewCache[fetchUrl] = preview;

                if (gen === fetchGeneration && currentLink === link) {
                    showPreview(link, preview);
                }
            })
            .catch(function () { });
    }

    document.addEventListener('pointerenter', function (e) {
        var link = e.target.closest('a[data-internal]');
        if (!link) return;

        currentLink = link;
        clearTimeout(hoverTimeout);
        hoverTimeout = setTimeout(function () {
            if (currentLink === link) {
                fetchPreview(link);
            }
        }, 300);
    }, true);

    document.addEventListener('pointerleave', function (e) {
        var link = e.target.closest('a[data-internal]');
        if (!link) return;
        hidePreview();
    }, true);

    // Also hide on click — covers the case where pointerleave never fires
    // because the link element gets removed by the outerHTML swap
    document.addEventListener('click', function (e) {
        var link = e.target.closest('a[data-internal]');
        if (link) {
            hidePreview();
        }
    }, true);

    // ── Browser back/forward ────────────────────────────────────

    window.addEventListener('popstate', function () {
        hidePreview();
    });

    // Also hide on page visibility change (tab switch, etc.)
    document.addEventListener('visibilitychange', function () {
        hidePreview();
    });

    // ── htmx navigation ──────────────────────────────────────────

    document.addEventListener('htmx:beforeSwap', function () {
        hidePreview();
    });

    document.addEventListener('htmx:afterSettle', function () {
        // Belt-and-suspenders: hide again after swap settles
        hidePreview();

        var content = document.getElementById('content');
        if (!content) return;

        if (window.Prism) {
            Prism.highlightAllUnder(content);
        }

        var charts = content.querySelectorAll('.mermaid');
        if (charts.length > 0) {
            if (window.mermaid) {
                mermaid.run({ nodes: charts });
            } else {
                var s = document.createElement('script');
                s.src = 'https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.9.1/mermaid.min.js';
                s.onload = function () {
                    mermaid.initialize({ startOnLoad: false });
                    mermaid.run({ nodes: charts });
                };
                document.head.appendChild(s);
            }
        }

        var title = content.getAttribute('data-title');
        if (title) {
            document.title = title;
        }

        window.scrollTo({ top: 0, behavior: 'instant' });

        // Re-init giscus if present (script tags from outerHTML swap don't execute)
        var giscusScript = content.querySelector('script[src*="giscus"]');
        if (giscusScript) {
            var fresh = document.createElement('script');
            for (var i = 0; i < giscusScript.attributes.length; i++) {
                fresh.setAttribute(
                    giscusScript.attributes[i].name,
                    giscusScript.attributes[i].value
                );
            }
            giscusScript.parentNode.replaceChild(fresh, giscusScript);
        }
    });
})();
