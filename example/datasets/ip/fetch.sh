#!/bin/sh
set -e

BASE_DIR="datasets/ip"
mkdir -p "$BASE_DIR"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

printf "[*] === START FETCHING & PREPARING IP DATASET (> 1,000,000 ITEMS) ===\n"

# ---------------------------------------------------------
# 1. FETCH & EXPAND MALICIOUS IPs (IPSUM & FIREHOL)
# ---------------------------------------------------------
printf "[*] Downloading malicious threat feeds (IPsum & FireHOL)...\n"
curl -sL "https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt" | \
    grep -v '^#' | awk '{print $1}' > "$TMP_DIR/ipsum_raw.txt" || true

curl -sL "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level1.netset" | \
    grep -v '^#' | grep -v '^$' >> "$TMP_DIR/firehol_raw.txt" || true

curl -sL "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level2.netset" | \
    grep -v '^#' | grep -v '^$' >> "$TMP_DIR/firehol_raw.txt" || true

printf "[*] Expanding CIDRs and deduplicating malicious IPs...\n"
awk '
function ip2dec(ip,   a) {
    split(ip, a, ".")
    return (a[1]*16777216) + (a[2]*65536) + (a[3]*256) + a[4]
}
function dec2ip(dec,   b1, b2, b3, b4) {
    b1 = int(dec / 16777216) % 256
    b2 = int(dec / 65536) % 256
    b3 = int(dec / 256) % 256
    b4 = dec % 256
    return b1 "." b2 "." b3 "." b4
}
{
    line = $1
    sub(/\r$/, "", line)
    if (line ~ /\//) {
        split(line, parts, "/")
        ip = parts[1]
        mask = parts[2] + 0
        if (mask >= 20 && mask <= 32) {
            span = 2 ^ (32 - mask)
            base = ip2dec(ip)
            for (i = 0; i < span; i++) {
                print dec2ip(base + i)
            }
        }
    } else if (line ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$/) {
        print line
    }
}
' "$TMP_DIR/ipsum_raw.txt" "$TMP_DIR/firehol_raw.txt" | sort -u | head -n 1000000 > "$TMP_DIR/malicious_ips.txt"

TOTAL_MALICIOUS=$(wc -l < "$TMP_DIR/malicious_ips.txt")
printf "[+] Extracted %s distinct malicious IPs.\n" "$TOTAL_MALICIOUS"

# ---------------------------------------------------------
# 2. GENERATE 1,000,000 CLEAN PUBLIC IPs (EXCLUDING BOGONS & MALICIOUS)
# ---------------------------------------------------------
printf "[*] Generating 1,000,000 valid, non-bogon public IPs for clean pool...\n"
awk -v target=1000000 '
BEGIN {
    srand(67890)
    count = 0
    while (count < target) {
        o1 = int(rand() * 223) + 1 # 1 to 223
        if (o1 == 10 || o1 == 127) continue
        o2 = int(rand() * 256)
        if (o1 == 172 && (o2 >= 16 && o2 <= 31)) continue
        if (o1 == 192 && o2 == 168) continue
        if (o1 == 169 && o2 == 254) continue
        o3 = int(rand() * 256)
        o4 = int(rand() * 254) + 1 # 1 to 254
        print o1 "." o2 "." o3 "." o4
        count++
    }
}
' > "$TMP_DIR/raw_benign_ips.txt"

# Lọc bỏ các IP trùng với danh sách đen
awk 'NR==FNR { bad[$1]=1; next } !bad[$1]' \
    "$TMP_DIR/malicious_ips.txt" "$TMP_DIR/raw_benign_ips.txt" | head -n 1000000 > "$TMP_DIR/benign_ips.txt"

TOTAL_BENIGN=$(wc -l < "$TMP_DIR/benign_ips.txt")
printf "[+] Generated %s clean public IPs.\n" "$TOTAL_BENIGN"

# ---------------------------------------------------------
# 3. SHUFFLE AND SPLIT (80% TRAIN / 20% TEST)
# ---------------------------------------------------------
printf "[*] Shuffling and splitting datasets (80%% Train / 20%% Test)...\n"

# Benign split (800k / 200k)
shuf "$TMP_DIR/benign_ips.txt" > "$TMP_DIR/shuf_benign.txt"
head -n 800000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/train_neg.txt"
tail -n 200000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/test_neg.txt"

# Malicious split (800k / 200k)
shuf "$TMP_DIR/malicious_ips.txt" > "$TMP_DIR/shuf_malicious.txt"
head -n 800000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/train_pos.txt"
tail -n 200000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/test_pos.txt"

printf "\n[+] IP DATASET READY AT: %s\n" "$BASE_DIR"
printf "    - Train Positives : %s lines\n" "$(wc -l < "$BASE_DIR/train_pos.txt")"
printf "    - Train Negatives : %s lines\n" "$(wc -l < "$BASE_DIR/train_neg.txt")"
printf "    - Test Positives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_pos.txt")"
printf "    - Test Negatives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_neg.txt")"
