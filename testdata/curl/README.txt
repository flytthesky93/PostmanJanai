Sample cURL commands for manual testing in PostmanJanai (Import → Request from cURL).

Usage: open a file, copy the single line, paste into the cURL import dialog.

Requires network access to https://httpbin.org (public test API).

Files:
  01_get_simple.curl          — GET, no body
  02_get_with_query.curl      — GET with query string in URL
  03_post_json.curl           — POST JSON body
  04_post_form.curl           — POST application/x-www-form-urlencoded
  05_get_capital_g.curl       — GET with -G: query built from -d parts
  06_post_multipart_text.curl — POST multipart (text fields only)
