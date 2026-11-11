#!/bin/sh
# scripts/fetch_datasets.sh - POSIX Compliant Dataset Ingestion Engine
# Thu thập, phân rã CIDR và chuẩn hóa dữ liệu cho SLBF
set -e

DATA_DIR="data"
mkdir -p "$DATA_DIR/ip" "$DATA_DIR/file" "$DATA_DIR/phone"

# ==============================================================================
# HÀM PHÂN RÃ CIDR BẰNG POSIX AWK (ZERO EXTERNAL DEPENDENCY)
# Chuyển đổi các dải mạng IPv4 (x.x.x.x/yy) và IP đơn lẻ thành danh sách IP độc lập
# ==============================================================================
expand_cidr() {
  awk '
    function ip2int(ip,   oct) {
      split(ip, oct, ".")
      return oct[1]*16777216 + oct[2]*65536 + oct[3]*256 + oct[4]
    }
    function int2ip(val,   o1, o2, o3, o4) {
      o1 = int(val / 16777216) % 256
      o2 = int(val / 65536) % 256
      o3 = int(val / 256) % 256
      o4 = int(val) % 256
      return o1 "." o2 "." o3 "." o4
    }
    {
      # Loại bỏ comment và khoảng trắng thừa
      sub(/#.*$/, "", $0)
      gsub(/[ \t\r]/, "", $0)
      if ($0 == "") next

      if (index($0, "/") == 0) {
        # Địa chỉ IP đơn lẻ
        if (split($0, oct, ".") == 4) {
          print $0
        }
      } else {
        # Dải mạng CIDR
        split($0, parts, "/")
        ip = parts[1]
        mask = parts[2] + 0

        # Giới hạn an toàn: không mở rộng các dải mạng quá lớn (nhỏ hơn /20)
        # để tránh làm tràn ổ đĩa (dải /20 sinh 4096 IPs)
        if (mask < 20) {
          mask = 20
        }

        count = 2 ^ (32 - mask)
        base = ip2int(ip)
        base = int(base / count) * count

        for (i = 0; i < count; i++) {
          print int2ip(base + i)
        }
      }
    }
  '
}

# ==============================================================================
# 1. DỮ LIỆU IP (models/ip.go - uint32)
# ==============================================================================
echo "[*] [1/3] Đang tải tập dữ liệu IP..."

# POSITIVES: Kết hợp Blocklist.de (~25.000 IP) và CINS Army (~15.000 IP)
echo "    -> Đang lấy danh sách IP độc hại (Blocklist.de & CINS Army)..."
(
  curl -sSL "https://lists.blocklist.de/lists/all.txt"
  curl -sSL "http://cinsscore.com/list/ci-badguys.txt"
) | expand_cidr | sort -u > "$DATA_DIR/ip/train_pos.txt"

# NEGATIVES: Phân rã 15 dải mạng của Cloudflare (curl đóng kết nối tức thì)
# Sắp xếp và lọc trùng toàn bộ trước khi trích xuất 50.000 IP sạch
echo "    -> Đang phân rã dải mạng Cloudflare sang IP sạch..."
curl -sSL "https://www.cloudflare.com/ips-v4" \
  | expand_cidr \
  | sort -u \
  | head -n 50000 > "$DATA_DIR/ip/train_neg.txt"

echo "[+] Dữ liệu IP hoàn tất:"
echo "    - Positives (Độc hại): $(wc -l < "$DATA_DIR/ip/train_pos.txt") địa chỉ IP"
echo "    - Negatives (Sạch)    : $(wc -l < "$DATA_DIR/ip/train_neg.txt") địa chỉ IP"

# ==============================================================================
# 2. DỮ LIỆU FILE HASH (models/file.go - [32]byte SHA-256)
# ==============================================================================
echo "[*] [2/3] Đang tải tập dữ liệu File Hash (SHA-256)..."

# POSITIVES: Tải toàn bộ luồng mã băm, lọc định dạng, đưa vào bộ đệm của sort -u
# curl sẽ nhận EOF và kết thúc hoàn toàn trước khi head nhận dữ liệu
echo "    -> Đang lấy mã băm SHA-256 mã độc từ kho IoC mở..."
curl -sSL "https://raw.githubusercontent.com/romainmarcoux/malicious-hash/main/full-hash-sha256-aa.txt" \
  | tr -d '\r' \
  | awk 'length($0)==64 && $0 ~ /^[0-9a-fA-F]{64}$/' \
  | sort -u \
  | head -n 50000 > "$DATA_DIR/file/train_pos.txt"

# NEGATIVES: Băm trực tiếp các tệp thực thi/thư viện sạch của hệ điều hành POSIX hiện tại
echo "    -> Đang tạo mã băm SHA-256 sạch từ các thư viện hệ thống..."
find /bin /usr/bin /lib /usr/lib -maxdepth 2 -type f 2>/dev/null \
  | xargs sha256sum 2>/dev/null \
  | awk '{print $1}' \
  | awk 'length($0)==64 && $0 ~ /^[0-9a-fA-F]{64}$/' \
  | sort -u \
  | head -n 50000 > "$DATA_DIR/file/train_neg.txt"

echo "[+] Dữ liệu File Hash hoàn tất:"
echo "    - Positives (Malware): $(wc -l < "$DATA_DIR/file/train_pos.txt") hashes"
echo "    - Negatives (Clean)  : $(wc -l < "$DATA_DIR/file/train_neg.txt") hashes"

# ==============================================================================
# 3. DỮ LIỆU SỐ ĐIỆN THOẠI (models/phone.go - string)
# ==============================================================================
echo "[*] [3/3] Đang tạo mẫu dữ liệu Số điện thoại..."

cat << 'EOF' > "$DATA_DIR/phone/train_pos.txt"
+84247770001
+84247770002
+84287771234
+84287775678
+84249998888
+88216123456
+88213654321
+22455112233
19001790
19001791
19006565
0247771234
0287775678
0249998888
EOF

cat << 'EOF' > "$DATA_DIR/phone/train_neg.txt"
111
112
113
114
115
18001090
18001091
18001560
18008098
18008168
1900545413
1900545415
02438257257
02838297373
EOF

echo "[+] Dữ liệu Số điện thoại hoàn tất:"
echo "    - Positives: $(wc -l < "$DATA_DIR/phone/train_pos.txt") số"
echo "    - Negatives: $(wc -l < "$DATA_DIR/phone/train_neg.txt") số"
