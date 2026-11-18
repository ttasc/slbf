#!/bin/sh
set -e

BASE_DIR="datasets/url"
mkdir -p "$BASE_DIR"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

printf "[*] === START FETCHING & PREPARING URL DATASET (> 1,000,000 ITEMS) ===\n"

# ---------------------------------------------------------
# 1. FETCH 1,000,000 BENIGN URLs (TRANCO TOP 1M)
# ---------------------------------------------------------
printf "[*] Downloading Tranco Top 1M benign domains...\n"
curl -sL "https://tranco-list.eu/top-1m.csv.zip" -o "$TMP_DIR/tranco.zip"

printf "[*] Extracting benign domains...\n"
unzip -p "$TMP_DIR/tranco.zip" | cut -d',' -f2 | tr -d '\r' | head -n 1000000 > "$TMP_DIR/raw_benign.txt"
TOTAL_BENIGN=$(wc -l < "$TMP_DIR/raw_benign.txt")
printf "[+] Extracted %s benign domains.\n" "$TOTAL_BENIGN"

# ---------------------------------------------------------
# 2. FETCH & GENERATE 1,000,000 MALICIOUS URLs
# ---------------------------------------------------------
printf "[*] Downloading active malware & phishing feeds...\n"
curl -sL "https://raw.githubusercontent.com/stamparm/maltrail/master/trails/static/malware/urlhaus.txt" \
    | grep -v '^#' | tr -d '\r' | grep -v '^$' > "$TMP_DIR/seed_malicious.txt" || true

TOTAL_SEEDS=$(wc -l < "$TMP_DIR/seed_malicious.txt")
printf "[+] Collected %s raw malicious seeds. Expanding to 1,000,000 items...\n" "$TOTAL_SEEDS"

# Sử dụng awk để kết hợp domain độc hại với các payload phishing/malware thực tế
awk -v target=1000000 '
BEGIN {
    srand(12345)
    paths[0] = "/login/verify.php?token="
    paths[1] = "/wp-content/plugins/fix/gate.php?id="
    paths[2] = "/secure/account-update/auth.cgi?session="
    paths[3] = "/banking/signin/index.html?ref="
    paths[4] = "/webscr/cmd_login_submit/dispatch="
    paths[5] = "/admin/upload/shell.php?cmd="
    paths[6] = "/api/v1/auth/session/renew?key="
    paths[7] = "/cpanel/reset-password/confirm?u="
    paths_count = 8
    count = 0
}
{
    sub(/^https?:\/\//, "", $0)
    sub(/\/.*$/, "", $0)
    if (length($0) > 3) {
        seeds[count++] = $0
    }
}
END {
    if (count == 0) {
        # Fallback seeds nếu mạng bị chặn
        seeds[0] = "phishing-bank-update.xyz"
        seeds[1] = "secure-login-attempt.top"
        seeds[2] = "malware-delivery-cdn.ru"
        seeds[3] = "verify-account-security.cc"
        count = 4
    }

    total = 0
    while (total < target) {
        dom = seeds[int(rand() * count)]
        p = paths[int(rand() * paths_count)]
        rand_token = sprintf("%x%x", int(rand() * 2147483647), int(rand() * 2147483647))
        print "http://" dom p rand_token
        total++
    }
}
' "$TMP_DIR/seed_malicious.txt" | head -n 1000000 > "$TMP_DIR/raw_malicious.txt"

TOTAL_MALICIOUS=$(wc -l < "$TMP_DIR/raw_malicious.txt")
printf "[+] Generated %s malicious URLs.\n" "$TOTAL_MALICIOUS"

# ---------------------------------------------------------
# 3. SHUFFLE AND SPLIT (80% TRAIN / 20% TEST)
# ---------------------------------------------------------
printf "[*] Shuffling and splitting datasets (80%% Train / 20%% Test)...\n"

# Benign split (800k / 200k)
shuf "$TMP_DIR/raw_benign.txt" > "$TMP_DIR/shuf_benign.txt"
head -n 800000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/train_neg.txt"
tail -n 200000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/test_neg.txt"

# Malicious split (800k / 200k)
shuf "$TMP_DIR/raw_malicious.txt" > "$TMP_DIR/shuf_malicious.txt"
head -n 800000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/train_pos.txt"
tail -n 200000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/test_pos.txt"

printf "\n[+] URL DATASET READY AT: %s\n" "$BASE_DIR"
printf "    - Train Positives : %s lines\n" "$(wc -l < "$BASE_DIR/train_pos.txt")"
printf "    - Train Negatives : %s lines\n" "$(wc -l < "$BASE_DIR/train_neg.txt")"
printf "    - Test Positives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_pos.txt")"
printf "    - Test Negatives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_neg.txt")"
