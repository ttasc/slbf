# ĐỀ CƯƠNG CHI TIẾT DỰ ÁN NGHIÊN CỨU & TRIỂN KHAI

**TÊN ĐỀ TÀI:** Nghiên cứu và xây dựng thư viện Learned Bloom Filter hiệu năng cao, mô-đun hóa bằng Go và tích hợp các mô hình học máy chuyên biệt.

**HƯỚNG NGHIÊN CỨU:** Cấu trúc dữ liệu mở rộng (Advanced Data Structures), Trí tuệ nhân tạo (Machine Learning) và Tối ưu hoá Hệ thống (System Optimization).

## I. LÝ DO CHỌN ĐỀ TÀI (ĐẶT VẤN ĐỀ)
Trong bối cảnh các hệ thống phân tán khối lượng lớn như API Gateway, phần mềm phản ứng mã độc (EDR), hay Web Application Firewall (WAF) phát triển, việc kiểm tra sự tồn tại của một bản ghi (ví dụ: IP bẩn, URL độc hại, File Hash, Revoked JWT) với số lượng hàng trăm triệu phần tử đang trở thành thách thức lớn về độ trễ cực thấp và chi phí phần cứng. Bộ lọc Bloom (Traditional Bloom Filter - TBF) luôn là lựa chọn hàng đầu nhờ khả năng tra cứu thời gian thực (O(1)) mà không cần lưu trực tiếp dữ liệu. Tuy nhiên, khi dung lượng dữ liệu tăng cao, TBF yêu cầu một lượng RAM tĩnh rất lớn (lên tới hàng Gigabytes) để duy trì tỷ lệ nhận diện nhầm (False Positive Rate - FPR) ở mức an toàn (<1%).

Vào năm 2018, khái niệm **Learned Bloom Filter (LBF)** được đề xuất, sử dụng Mô hình học máy (Machine Learning) đóng vai trò là "màng lọc cấp 1" nhằm thay thế một phần cấu trúc băm truyền thống, giúp tiết kiệm lên tới 50-80% lượng bộ nhớ tiêu thụ. Tuy nhiên, hiện tại các bản triển khai mã nguồn mở của LBF tồn tại những nhược điểm:
1. Thiếu tính đóng gói, Mô hình AI và Thư viện Lõi (Core structure) thường bị kết dính với nhau (Tight Coupling), gây khó khăn trong việc tùy biến.
2. Tại môi trường sử dụng Garbage Collection như Go, quá trình chạy suy luận AI dễ sinh ra bộ nhớ động trên Heap, dẫn đến độ trễ ngắt quãng (GC Pause) làm tê liệt hệ thống thời gian thực.

Xuất phát từ thực tiễn trên, dự án này sẽ **xây dựng một thư viện Learned Bloom Filter đột phá bằng ngôn ngữ Go (đặt tên dự án là SLBF - Simple Learned Bloom Filter)**. Dự án tuân thủ nghiêm ngặt triết lý thiết kế tối giản (*Suckless*), tính nguyên bản của ngôn ngữ (*The Go Way*), tách biệt hoàn toàn phần lõi với các mô hình AI. Thư viện sẽ nhắm đến giới hạn cao nhất về mặt hiệu năng xử lý cực hạn (Zero-Allocation Memory).

## II. MỤC TIÊU NGHIÊN CỨU VÀ TRIỂN KHAI
1. **Mục tiêu học thuật:** Làm rõ cơ sở toán học và cơ chế tối ưu dung lượng của Learned Bloom Filter (Tỉ lệ âm bản/False Positive Rate, Hàm băm...).
2. **Mục tiêu thực tiễn:** Sẽ xây dựng một thư viện phần mềm bằng ngôn ngữ Go:
  - **Thiết kế Lõi Cấu trúc (Core Architecture):** Xây dựng một Engine Bloom Filter cốt lõi, không phụ thuộc kiểu dữ liệu (Data-agnostic) dựa trên Generics của Go 1.18+.
  - **Nguyên tắc "Go Way":** Định nghĩa tiêu chuẩn Interface khắt khe, biến thư viện thành một bộ khung mở (Framework), cho phép cắm (plug-in) các Mô hình AI bất kỳ vào hệ thống mà không cần sửa đổi mã nguồn gốc (Open/Closed Principle).
  - **Hiệu suất & An toàn Đa luồng:** Giải quyết bài toán nâng cấp mô hình theo thời gian thực (Atomic Swapping) với tiêu chí: *Không gián đoạn luồng tra cứu (Zero-Downtime)* và *Không thất thoát tập dữ liệu cũ (Zero False Negatives)*. Vận dụng Zero-Allocation cho luồng thực thi dữ liệu nóng.
  - **Triển khai ứng dụng thực tế (Proof-of-Concept):** Sẽ triển khai, chứng minh tính đúng đắn trên bài toán *Tra cứu URL độc hại (Malicious URL Filtering)*, đồng thời xây dựng công cụ dòng lệnh Benchmark để đo đạc và đưa ra kết quả so sánh khách quan với hệ thống truyền thống. Lên thiết kế và ý tưởng (Mocking) sẵn sàng mở rộng các bài toán khác (Token, IPs, Hash) ở tương lai.

## III. ĐỐI TƯỢNG VÀ PHẠM VI NGHIÊN CỨU
- **Đối tượng nghiên cứu:**
  - Lý thuyết cấu trúc dữ liệu và giải thuật (Mảng băm, Mảng nhị phân Bitset).
  - Thuật toán hồi quy và mô hình Neural nhỏ tích hợp (Lightweight Embeddable Models).
  - Kỹ thuật lập trình và quản lý bộ nhớ của Golang (Go Generics, Pointers, Garbage Collection optimization).
- **Phạm vi triển khai:**
  - Tập trung giải quyết bài toán Filter đối chiếu cấu trúc dữ liệu cấp phần tử. Dự kiến cài đặt và tích hợp cho 5 dạng mẫu mô hình đại diện bao gồm: Phishing URL (URLClassifier), C&C Server IP (IPClassifier), Spam Email Domain, API Token JWT (TokenClassifier) và Malware Hash (FileHashClassifier).
  - Đề tài tập trung tối ưu quá trình Query/Inference. Các quá trình Training cho thuật toán AI (Epoch/Learning Rate) có thể cấu hình động thông qua framework mà không phụ thuộc vào các engine đồ sộ (như TensorFlow hay PyTorch).

## IV. PHƯƠNG PHÁP NGHIÊN CỨU
- **Phương pháp lý thuyết:** Đọc và tổng hợp các luận văn, báo cáo về Kraska LBF (2018), cách tối ưu hàm băm Non-Cryptographic (như FNV-1a, Bit shift algorithm) dùng trong môi trường HFT (High-Frequency Trading).
- **Phương pháp thực nghiệm (Empirical Method):**
  - Lập trình kiến trúc nguyên mẫu theo triết lý "Interface-Driven Design".
  - Đóng gói toàn bộ tiêu chuẩn mô hình thông qua interface chung, từ đó thư viện hệ thống chỉ việc nhúng module tùy chọn tương ứng.
  - Sẽ xây dựng hệ thống mô phỏng Benchmarking tool bằng lệnh CLI để trực tiếp đẩy lượng Request mẫu vào (stress test), sau đó dùng module `runtime.MemStats` đo kiểm sát sườn RAM đã cấp phát, TPS (Transaction Per Second), FPR (False Positive Rate) trên các bộ dữ liệu Train / Unseen Test tách biệt.

## V. DỰ KIẾN CẤU TRÚC BÁO CÁO (NỘI DUNG NGHIÊN CỨU CHI TIẾT)

**CHƯƠNG 1: TỔNG QUAN VẤN ĐỀ VÀ CƠ SỞ LÝ THUYẾT**

1.1 Hạn chế của Bloom Filter trong kỹ nghệ Dữ liệu lớn (Big Data).

1.2 Cơ sở toán học của Learned Bloom Filter (Tương tác bù trừ giữa Ngưỡng AI Threshold và Bộ nhớ bitset phụ trợ - Fallback Bitset).

1.3 Vai trò của bộ đệm siêu nhẹ và kỹ thuật tối ưu hóa Memory Layout trong môi trường Go.

**CHƯƠNG 2: THIẾT KẾ KIẾN TRÚC THƯ VIỆN LÕI SLBF**

2.1 **Kiến trúc Modular qua Generic & Interface:** Sẽ định nghĩa khuôn dạng chặt chẽ bằng cấu trúc Generic `Filter[T]` tương tác đa kiểu (chuỗi, số lượng). Đề xuất mẫu bản phân tách các API thông qua định nghĩa cấu trúc độc lập như interface `LearnedModel`.

2.2 **Tối ưu Hóa Truy Suất đồng thời (Concurrency Readiness):** Thiết kế lock pattern (`sync.RWMutex`) hỗ trợ hệ thống chỉ đọc và kỹ thuật Swap AI runtime cho phép update không cần re-start máy chủ.

2.3 **Nguyên lý Serialization/Hot load (Save/Restore LBF data):** Dự định ứng dụng các Protocol nhị phân, marshaller và bộ giải nén chuẩn, đáp ứng phục hồi toàn khối SLBF snapshot cực nhanh trên client thay vì sinh bảng nội mạng lại từ đầu.

**CHƯƠNG 3: TRIỂN KHAI VÀ XÂY DỰNG MÔ HÌNH HỌC MÁY CHUYÊN BIỆT**

Nhằm chứng minh cho khả năng cắm mở linh hoạt (Pluggable), đề tài tiến hành code, huấn luyện và thiết lập zero-allocation các hàm (Predict, Hash) thuộc gói riêng cho các Model:

3.1 Nhận dạng Domain của một chuỗi liên kết – (Cho bảo vệ Anti-Phishing Web `URLClassifier`)

3.2 Kiểm lọc nhanh bộ mẫu cấp chuỗi số – (`IPClassifier` cấp Gateway & API)

3.3 Giải thuật Băm Chuỗi Mật độ Hữu Dụng (`TokenClassifier`)

3.4 Sáng chế Mô hình AI đánh giá rủi ro nhanh dành cho EDR qua hệ số byte nhị phân thô (`FileHashClassifier`).

**CHƯƠNG 4: THỰC NGHIỆM VÀ KẾT QUẢ ĐÁNH GIÁ ĐO LƯỜNG**
4.1 Chuẩn bị số liệu mẫu tập train (dữ liệu Độc - data Pos / dữ liệu An Toàn - data Neg) và tập kiểm thử bù đắp ngoại bào chưa gặp.
4.2 Công tác triển khai chạy Stress-test so chiếu chênh lệch độ chính xác so với truyền thống. Phân bổ công tác viết cli ứng dụng (`builder` commandline).
4.3 Trình diễn Số lượng/Dữ liệu Báo Cáo. Phân tích điểm thu nạp: Hiệu số tiết kiệm Ram Saved đo kiểm KB Heap alloc; Tốc Độ Băng Thông Req/s (TPS Benchmark Analytics). Đánh giá xem đã hoàn thành mục tiêu tiết kiệm chưa.

**KẾT LUẬN & HƯỚNG PHÁT TRIỂN TIẾP THEO**

## VI. TIẾN ĐỘ THỰC HIỆN DỰ KIẾN
1. **Giai đoạn 1 (Tuần 1 - 2):** Nghiên cứu kiến trúc tổng quan cấu trúc bộ nhớ Bloom Filter, viết prototype thuật toán nhị phân, hoàn thành luồng TBF chuẩn hoá. Khai triển Go interface.
2. **Giai đoạn 2 (Tuần 3 - 4):** Tiến hành tích hợp ML cho phần Core bằng Generic Generics 1.20+ (Thiết kế `Config[T]`, cơ chế đa luồng Mutex RLock SwapModel an toàn 100%).
3. **Giai đoạn 3 (Tuần 5 - 6):** Triển khai xây dựng cụm Data Plugin. Trực tiếp thực thi bài toán phân tích n-gram cấu trúc nhận dạng mô hình chặn Web Độc, Web giả mạo URLs làm ví dụ chuẩn hoá Code Architecture. Triển khai các kĩ thuật loại trừ RAM Alloc.
4. **Giai đoạn 4 (Tuần 7):** Phát triển CLI Tools `benchmark, build, test` phục vụ việc truy thu Log Real-time đánh giá chất lượng hệ thống phân phối mô hình AI Offline chuẩn Binary Encoding cho thư viện, tuân thủ không lặp code UI. Đánh giá Mock Metrics cuối cùng.
5. **Giai đoạn 5 (Tuần 8):** Phân tích đo đạc Throughput và Tỷ lệ chặn nhầm (FPR) xuất từ tập file dữ liệu lớn. Tinh chỉnh, Review và báo cáo đề tài. Định hướng lộ trình Open-source các kho `token.go, ip.go...`

## VII. DỰ KIẾN KẾT QUẢ ĐẠT ĐƯỢC
Kết thúc đề tài, đồ án dự kiến sẽ tạo ra và minh chứng được các chỉ số và kết quả vật lý gồm:
1. Cấu trúc Source Code Package (Golang) độc lập phát hành mở public hoàn toàn Module, đáp ứng nguyên tắc thiết kế mẫu Mẫu Strategy, Abstract Plugin cực hẹp để tái lập vòng cung AI Bloom System tại cấp Gateway doanh nghiệp phân tán.
2. Hệ thống Benchmark Command Line Utility thể hiện chỉ số vượt ngưỡng bằng Báo cáo đánh giá: Chứng minh công nghệ tối ưu tiết kiệm ~> **50%-75% không gian Tĩnh (Memory Saving Footprints)**, tối đa sức chống tải O(1) CPU Throughputs ngang bằng hoặc >100% khi đem đối chiếu chéo (so TBF bitsets kích cỡ tiêu chuẩn FPR/Unseen FPR Testing Logs Analysis).

## VIII. Danh mục tài liệu tham khảo

`[1] B. H. Bloom, “Space/time trade-offs in hash coding with allowable errors,” Communications of the ACM, vol. 13, no. 7, pp. 422–426, 1970.`

`[2] M. Mitzenmacher, “A Model for Learned Bloom Filters and Related Structures,” arXiv preprint arXiv:1802.00884, 2018.`

`[3] T. Kraska, A. Beutel, E. H. Chi, J. Dean, and N. Polyzotis, “The case for learned index structures,” in Proceedings of the 2018 International Conference on Management of Data (SIGMOD '18), pp. 489–504, 2018.`

`[4] P. Dai et al., "A Learned Bloom Filter Framework for Big Data Network Security," IEEE Access, vol. 7, pp. 106095-106103, 2019.`

`[5] M. Mitzenmacher, "Optimizing Learned Bloom Filters for Realistic Configurations," arXiv preprint arXiv:1902.04012, 2019.`

`[6] S. Abu-Nimeh, D. Nappa, X. Wang, and S. Nair, “A comparison of machine learning techniques for phishing detection,” in Proceedings of the anti-phishing working groups 2nd annual eCrime researchers summit, pp. 60–69, 2007.`

`[7] S. Geravand and M. Ahmadi, "Bloom filter applications in network security: A state-of-the-art survey," Computer Networks, vol. 57, no. 15, pp. 3205-3224, 2013.`

`[8] L. C. Noll, "Fowler-Noll-Vo hash (FNV)," IETF Draft, 2015. [Online]. Available: http://www.isthe.com/chongo/tech/comp/fnv/`

`[9] R. Cox et al., "The Go Memory Model," Go Documentation, 2024. [Online]. Available: https://golang.org/ref/mem`

