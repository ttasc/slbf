#!/bin/sh
set -e

BASE_DIR="."
mkdir -p "$BASE_DIR"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

printf "[*] === START FETCHING & PREPARING USER-AGENT DATASET (> 1,000,000 ITEMS) ===\n"

# ---------------------------------------------------------
# 1. FETCH & GENERATE 1,000,000 BENIGN BROWSER USER-AGENTS
# ---------------------------------------------------------
printf "[*] Generating 1,000,000 legitimate browser User-Agents (Chrome, Firefox, Safari, Edge)...\n"

awk -v target=1000000 '
BEGIN {
    srand(112233)

    # Danh sách hệ điều hành máy tính và di động
    os_list[0] = "Windows NT 10.0; Win64; x64"
    os_list[1] = "Windows NT 11.0; Win64; x64"
    os_list[2] = "Macintosh; Intel Mac OS X 10_15_7"
    os_list[3] = "Macintosh; Intel Mac OS X 13_4_1"
    os_list[4] = "X11; Linux x86_64"
    os_list[5] = "X11; Ubuntu; Linux x86_64"
    os_list[6] = "Linux; Android 13; SM-S908B"
    os_list[7] = "iPhone; CPU iPhone OS 16_5 like Mac OS X"
    os_count = 8

    count = 0
    while (count < target) {
        os = os_list[int(rand() * os_count)]
        browser_type = int(rand() * 4)
        c_ver = 110 + int(rand() * 25) # Chrome 110-134
        b_num = 5000 + int(rand() * 1500)
        p_num = 10 + int(rand() * 180)

        if (browser_type == 0) {
            # Chrome
            print "Mozilla/5.0 (" os ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" c_ver ".0." b_num "." p_num " Safari/537.36"
        } else if (browser_type == 1) {
            # Edge
            print "Mozilla/5.0 (" os ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" c_ver ".0." b_num "." p_num " Safari/537.36 Edg/" c_ver ".0." b_num "." p_num
        } else if (browser_type == 2) {
            # Firefox
            ff_ver = 105 + int(rand() * 25)
            print "Mozilla/5.0 (" os "; rv:" ff_ver ".0) Gecko/20100101 Firefox/" ff_ver ".0"
        } else {
            # Safari
            print "Mozilla/5.0 (" os ") AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.4 Safari/605.1.15"
        }
        count++
    }
}
' > "$TMP_DIR/raw_benign_ua.txt"

TOTAL_BENIGN=$(wc -l < "$TMP_DIR/raw_benign_ua.txt")
printf "[+] Generated %s benign browser User-Agents.\n" "$TOTAL_BENIGN"

# ---------------------------------------------------------
# 2. FETCH & GENERATE 1,000,000 MALICIOUS BOTS / SCRAPERS
# ---------------------------------------------------------
printf "[*] Downloading bad-bot seed lists from threat feeds...\n"
curl -sL "https://raw.githubusercontent.com/mitchellkrogza/nginx-ultimate-bad-bot-blocker/master/_generator_lists/bad-user-agents.list" \
    | grep -v '^#' | tr -d '\r' | grep -v '^$' > "$TMP_DIR/seed_bots.txt" || true

TOTAL_SEEDS=$(wc -l < "$TMP_DIR/seed_bots.txt")
printf "[+] Collected %s seed bots. Expanding into 1,000,000 malicious UA variants...\n" "$TOTAL_SEEDS"

awk -v target=1000000 '
BEGIN {
    srand(445566)

    # Danh sách công cụ quét mã độc và script tự động
    tools[0] = "sqlmap"
    tools[1] = "nikto"
    tools[2] = "nmap"
    tools[3] = "masscan"
    tools[4] = "wpscan"
    tools[5] = "dirbuster"
    tools[6] = "gobuster"
    tools[7] = "python-requests"
    tools[8] = "aiohttp"
    tools[9] = "Go-http-client"
    tools[10] = "curl"
    tools[11] = "Wget"
    tools[12] = "Scrapy"
    tools[13] = "HeadlessChrome"
    tools[14] = "PhantomJS"
    tools[15] = "zgrab"
    tools_count = 16

    seed_count = 0
}
{
    sub(/\r$/, "", $0)
    if (length($0) > 2) {
        seeds[seed_count++] = $0
    }
}
END {
    count = 0
    while (count < target) {
        mode = int(rand() * 3)
        if (mode == 0 && seed_count > 0) {
            # Biến thể từ hạt giống tải về
            s = seeds[int(rand() * seed_count)]
            v_major = int(rand() * 5) + 1
            v_minor = int(rand() * 20)
            print s "/" v_major "." v_minor
        } else if (mode == 1) {
            # Biến thể tool tấn công & quét lỗ hổng
            t = tools[int(rand() * tools_count)]
            v1 = int(rand() * 9) + 1
            v2 = int(rand() * 30)
            v3 = int(rand() * 10)
            print t "/" v1 "." v2 "." v3
        } else {
            # Giả mạo bot crawler không hợp lệ
            bot_names[0] = "AutoScraper"
            bot_names[1] = "DataMiner"
            bot_names[2] = "VulnerabilityScanner"
            bot_names[3] = "ExploitTester"
            bot_names[4] = "EmailCollector"
            b = bot_names[int(rand() * 5)]
            print b "/v" (int(rand() * 5) + 1) ".0 (compatible; security-audit; +" sprintf("%x", int(rand()*1000000)) ")"
        }
        count++
    }
}
' "$TMP_DIR/seed_bots.txt" | head -n 1000000 > "$TMP_DIR/raw_malicious_ua.txt"

TOTAL_MALICIOUS=$(wc -l < "$TMP_DIR/raw_malicious_ua.txt")
printf "[+] Generated %s malicious User-Agents.\n" "$TOTAL_MALICIOUS"

# ---------------------------------------------------------
# 3. SHUFFLE AND SPLIT (80% TRAIN / 20% TEST)
# ---------------------------------------------------------
printf "[*] Shuffling and splitting datasets (80%% Train / 20%% Test)...\n"

# Benign split (800k / 200k)
shuf "$TMP_DIR/raw_benign_ua.txt" > "$TMP_DIR/shuf_benign.txt"
head -n 800000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/train_neg.txt"
tail -n 200000 "$TMP_DIR/shuf_benign.txt" > "$BASE_DIR/test_neg.txt"

# Malicious split (800k / 200k)
shuf "$TMP_DIR/raw_malicious_ua.txt" > "$TMP_DIR/shuf_malicious.txt"
head -n 800000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/train_pos.txt"
tail -n 200000 "$TMP_DIR/shuf_malicious.txt" > "$BASE_DIR/test_pos.txt"

printf "\n[+] USER-AGENT DATASET READY AT: %s\n" "$BASE_DIR"
printf "    - Train Positives : %s lines\n" "$(wc -l < "$BASE_DIR/train_pos.txt")"
printf "    - Train Negatives : %s lines\n" "$(wc -l < "$BASE_DIR/train_neg.txt")"
printf "    - Test Positives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_pos.txt")"
printf "    - Test Negatives  : %s lines\n" "$(wc -l < "$BASE_DIR/test_neg.txt")"
