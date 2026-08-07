# Tidekeepers — Keeper Catalog

**Tài liệu:** Danh sách và mô tả 10 Keeper cho MVP  
**Ngôn ngữ:** Tiếng Việt  
**Trạng thái:** Bản thiết kế đề xuất  
**Phạm vi:** Keeper, basket tham chiếu, vai trò, kỹ năng và định hướng cân bằng

---

## 1. Mục đích tài liệu

Tài liệu này mô tả 10 Keeper được xây dựng từ các nhóm tài sản số phổ biến. Mỗi Keeper là một nhân vật fantasy đại diện cho một kiểu hành vi thị trường, một phong cách chơi và một vai trò chiến thuật riêng.

Thiết kế tuân theo nguyên tắc:

```text
Coin hoặc tài sản thật
→ Basket dữ liệu
→ Current trong game
→ Keeper Market Performance
→ Fleet Performance
→ Daily Score
```

Keeper không đại diện trực tiếp cho một coin duy nhất. Mỗi Keeper sử dụng một basket gồm nhiều thành phần để:

- Giảm tác động của một tài sản tăng hoặc giảm bất thường.
- Tránh biến gameplay thành lựa chọn coin trực tiếp.
- Giữ được bản sắc fantasy của Tidekeepers.
- Dễ cân bằng và thay đổi dữ liệu trong các phiên bản sau.
- Giảm phụ thuộc vào tên, logo và thương hiệu bên ngoài.

Tên coin và trọng số basket chỉ nên xuất hiện trong hệ thống quản trị hoặc trang minh bạch dữ liệu. Giao diện gameplay chính chỉ hiển thị tên Current, Keeper, Sector và Role.

---

## 2. Quy ước chung

### 2.1. Thuộc tính Keeper

Mỗi Keeper có các thuộc tính cốt lõi:

```text
Name
Rarity
Sector
Role
Base Risk
Expected Turbulence
Passive Skill
Upgrade Path
Tags
Basket Mapping Version
```

### 2.2. Cách sử dụng basket

Mỗi basket phải được version hóa.

Ví dụ:

```yaml
keeperBasketVersion: crest-large-cap-v1
effectiveFrom: 2026-08-01
components:
  BTC: 0.35
  ETH: 0.25
  BNB: 0.15
  SOL: 0.15
  XRP: 0.10
```

Không được thay đổi thành phần hoặc trọng số basket trong khi một Tide đang chạy.

### 2.3. Ghi chú cân bằng

Các basket và trọng số trong tài liệu này là cấu hình thiết kế ban đầu. Trước khi triển khai, đội phát triển cần:

- Kiểm tra coverage dữ liệu của từng thành phần.
- Chuẩn hóa theo Expected Turbulence.
- Giới hạn ảnh hưởng của outlier.
- Kiểm tra tương quan giữa các basket.
- Tránh để hai Keeper có hành vi dữ liệu gần như giống nhau.
- Tách rõ Market Performance khỏi Skill Effect.

---

# 3. Danh sách Keeper

## 3.1. Crest Sovereign

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Crest |
| Role | Vanguard |
| Rarity | Common |
| Base Risk | Thấp–trung bình |
| Nhóm tham chiếu | Layer 1 vốn hóa lớn |
| Current hiển thị | Sovereign Current |

### Basket tham chiếu

```text
BTC 35%
ETH 25%
BNB 15%
SOL 15%
XRP 10%
```

### Fantasy

Crest Sovereign là một sư tử biển mặc giáp vàng, cầm tấm khiên khắc hình vương miện thủy triều. Nhân vật đại diện cho những dòng chảy lớn, bền vững và có ảnh hưởng rộng lên toàn thế giới Tidekeepers.

### Passive — Crown of the Tide

```text
Nếu ít nhất 3 thành phần trong basket vượt World Tide,
nhận +20% Global Relative Score.

Trong Broad Decline,
giảm 15% Depth Penalty của bản thân.
```

### Phong cách chơi

- Keeper cân bằng giữa tăng trưởng và phòng thủ.
- Phù hợp làm nhân vật khởi đầu.
- Hoạt động tốt khi các dòng lớn cùng di chuyển theo một hướng.
- Có khả năng giữ điểm tốt hơn các Growth Keeper trong ngày giảm mạnh.

### Điểm mạnh

- Ổn định trong nhiều Market Regime.
- Dễ hiểu với người chơi mới.
- Có thể làm trụ cột cho Crest Synergy.
- Không phụ thuộc quá nhiều vào một thành phần basket.

### Điểm yếu

- Khó tạo điểm bùng nổ.
- Có thể kém hiệu quả trong Rotation mạnh.
- Basket đa dạng làm giảm khả năng tận dụng một Sector dẫn đầu duy nhất.

### Tag đề xuất

```text
Stable
Large Current
Frontline
Relative Performance
```

### Hướng nâng cấp

#### Deep Crown

```text
Tăng hiệu quả giảm Depth Penalty từ 15% lên 25%.
```

#### Rising Throne

```text
Nếu Sovereign Current dẫn đầu Crest Sector,
nhận thêm Sector Relative Score.
```

---

## 3.2. Harbor Warden

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Harbor |
| Role | Warden |
| Rarity | Common |
| Base Risk | Rất thấp |
| Nhóm tham chiếu | Stablecoin |
| Current hiển thị | Harbor Current |

### Basket tham chiếu

```text
USDT 30%
USDC 30%
USDS 15%
DAI 15%
USDe 10%
```

### Fantasy

Harbor Warden là một thủy thủ rái cá mang khiên mỏ neo và chiếc đèn cảng không bao giờ tắt. Nhân vật không cố gắng đi nhanh nhất, mà đảm bảo Fleet không bị cuốn xuống đáy trong những ngày khắc nghiệt.

### Metric đặc biệt

Harbor Warden không nên được chấm chủ yếu bằng raw return. Keeper sử dụng metric ổn định riêng:

```text
Harbor Stability
= Peg Accuracy
+ Liquidity Stability
- Maximum Peg Deviation
```

### Passive — Unbroken Peg

```text
Mỗi ngày bảo vệ Keeper có Depth lớn nhất,
giảm 35% Depth Penalty của Keeper đó.

Nếu Harbor Stability đạt ngưỡng,
giảm thêm Hull Damage của Fleet.
```

### Phong cách chơi

- Keeper phòng thủ cốt lõi.
- Phù hợp với build Survival, Preserve và Control Risk.
- Có giá trị cao trong Broad Decline hoặc Shock.
- Không tạo nhiều Performance Score trong ngày tăng mạnh.

### Điểm mạnh

- Giảm biến động kết quả của Fleet.
- Bảo vệ Keeper có rủi ro cao.
- Tương tác tốt với Trickster và Growth Keeper.
- Hữu ích trong các Boss gây Pressure hoặc Depth Penalty.

### Điểm yếu

- Performance Ceiling thấp.
- Có thể làm Fleet thiếu sức tấn công.
- Giá trị giảm trong ngày Broad Growth ổn định.

### Tag đề xuất

```text
Defense
Stability
Hull Protection
Depth Control
```

### Hướng nâng cấp

#### Deep Shelter

```text
Tăng mức giảm Depth Penalty từ 35% lên 45%.
```

#### Safe Yield

```text
Nếu Fleet hoàn thành Daily Objective,
Harbor Warden nhận thêm một lượng nhỏ Risk-adjusted Score.
```

---

## 3.3. Iron Broker

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Forge |
| Role | Compounder |
| Rarity | Rare |
| Base Risk | Trung bình |
| Nhóm tham chiếu | Centralized Exchange Token |
| Current hiển thị | Commerce Current |

### Basket tham chiếu

```text
BNB 30%
WBT 20%
LEO 20%
CRO 15%
OKB 15%
```

### Fantasy

Iron Broker là một thương nhân cua thép với nhiều cánh tay, mỗi tay giữ một sổ cái, chìa khóa kho hoặc đồng xu phát sáng. Nhân vật đại diện cho sức mạnh tích lũy từ thanh khoản, giao dịch và hoạt động kinh tế kéo dài.

### Passive — Fee Engine

```text
Mỗi ngày liên tiếp Iron Broker được giữ trong Fleet
và Forge basket vượt benchmark:
nhận 1 stack Commerce.

Mỗi stack:
+5% Risk-adjusted Score.

Tối đa 4 stack.
```

### Phong cách chơi

- Keeper mạnh dần khi được giữ lâu trong đội hình.
- Thưởng cho người chơi cam kết với build thay vì thay Keeper liên tục.
- Phù hợp Voyage dài và build kinh tế.

### Điểm mạnh

- Scaling tốt qua nhiều ngày.
- Tương tác mạnh với Relic thưởng Keeper được giữ lâu.
- Có khả năng tạo điểm ổn định nếu Forge Current duy trì tốt.

### Điểm yếu

- Mất toàn bộ Commerce khi bị đưa khỏi Fleet.
- Khởi đầu yếu hơn Growth Keeper trực tiếp.
- Dễ bị ảnh hưởng trong Liquidity Shock.

### Tag đề xuất

```text
Economy
Scaling
Commitment
Forge
```

### Hướng nâng cấp

#### Deep Ledger

```text
Tăng giới hạn Commerce từ 4 lên 6 stack.
```

#### Liquid Reserve

```text
Khi Market Regime là Shock,
giảm một phần Pressure Penalty theo số Commerce hiện có.
```

---

## 3.4. Current Weaver

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Current |
| Role | Support |
| Rarity | Common |
| Base Risk | Trung bình–cao |
| Nhóm tham chiếu | Decentralized Exchange |
| Current hiển thị | Exchange Current |

### Basket tham chiếu

```text
HYPE 30%
UNI 25%
JUP 15%
CAKE 15%
CRV 15%
```

### Fantasy

Current Weaver là một pháp sư sứa điều khiển các dải nước phát sáng nối nhiều dòng chảy với nhau. Nhân vật không nhất thiết dẫn đầu về điểm cá nhân, nhưng có khả năng chuyển hiệu suất của nhiều Sector thành sức mạnh cho toàn Fleet.

### Passive — Flow Transfer

```text
Nếu Exchange Current và ít nhất một Sector khác
cùng vượt benchmark:

+15% Diversification Score cho toàn Fleet.
```

### Phong cách chơi

- Support dành cho đội hình đa Sector.
- Thưởng cho người chơi đọc Rotation và tạo Fleet linh hoạt.
- Hoạt động tốt khi nhiều nhóm cùng có sức mạnh tương đối.

### Điểm mạnh

- Kích hoạt Diversified Fleet dễ hơn.
- Tăng giá trị cho đội hình không tập trung một Sector.
- Phù hợp với Objective Diversify và Rotation Day.

### Điểm yếu

- Yếu khi volume thấp hoặc Calm Waters.
- Không có nhiều khả năng tự bảo vệ.
- Hiệu quả giảm nếu Fleet chỉ sử dụng một hoặc hai Sector.

### Tag đề xuất

```text
Support
Diversification
Rotation
Flow
```

### Hướng nâng cấp

#### Cross Current

```text
Passive kích hoạt nếu hai Sector bất kỳ vượt benchmark,
không bắt buộc Exchange Current phải là một trong hai.
```

#### Deep Liquidity

```text
Nếu Volume Trend của Exchange Current dương,
tăng thêm Skill Effect cho toàn Fleet.
```

---

## 3.5. Echo Seer

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Echo |
| Role | Oracle |
| Rarity | Rare |
| Base Risk | Trung bình |
| Nhóm tham chiếu | Oracle và dữ liệu |
| Current hiển thị | Oracle Current |

### Basket tham chiếu

```text
LINK 45%
PYTH 20%
RED 15%
TRB 10%
API3 10%
```

### Fantasy

Echo Seer là một con cú biển mù có thể nhìn thấy dòng chảy thông qua các vòng sóng âm. Nhân vật không dự đoán chắc chắn tương lai, nhưng giúp người chơi nhận được thêm dữ liệu để đưa ra quyết định có cơ sở.

### Passive — Read the Wake

```text
Trước khi khóa Fleet,
hiển thị thêm một Signal định lượng.
```

Signal bổ sung có thể thuộc một trong các loại:

```text
Sector Breadth
Correlation
Volume Trend
Turbulence
Depth Risk
Recovery Strength
```

### Hiệu ứng settlement

```text
Nếu Signal được hiển thị thuộc metric
thực sự tác động mạnh nhất trong ngày:

+10% Daily Objective contribution.
```

### Phong cách chơi

- Keeper thiên về thông tin và ra quyết định.
- Phù hợp với người chơi thích đọc Signal.
- Giúp giảm độ mơ hồ trước khi khóa Fleet.

### Điểm mạnh

- Tăng chất lượng quyết định trước Tide.
- Hữu ích trong Rotation, Sideways và ngày Signal phức tạp.
- Có thể giúp người chơi chọn Strategy phù hợp hơn.

### Điểm yếu

- Market Score cá nhân không cao.
- Sức mạnh phụ thuộc vào khả năng tận dụng thông tin của người chơi.
- Bị giảm hiệu quả trong Boss làm nhiễu Signal.

### Tag đề xuất

```text
Oracle
Information
Signal
Preparation
```

### Hướng nâng cấp

#### Clear Echo

```text
Hiển thị độ tin cậy của Signal bổ sung.
```

#### Stored Wake

```text
Cho phép giữ lại một Signal từ ngày trước
để so sánh với Signal hiện tại.
```

---

## 3.6. Veil Reversalist

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Veil |
| Role | Contrarian |
| Rarity | Rare |
| Base Risk | Cao |
| Nhóm tham chiếu | Privacy Coin |
| Current hiển thị | Veiled Current |

### Basket tham chiếu

```text
ZEC 30%
XMR 30%
DCR 15%
ARRR 15%
XVG 10%
```

### Fantasy

Veil Reversalist là một sát thủ mực đen có thể biến mất dưới đáy biển và xuất hiện trở lại khi cơn sóng bắt đầu đảo chiều. Nhân vật mạnh nhất khi thị trường trải qua cú giảm sâu rồi hồi phục.

### Passive — From the Depths

```text
Nếu Veiled Current giảm sâu hơn Expected Turbulence,
nhưng phục hồi ít nhất 50% từ đáy:

nhân đôi Resurface Score.
```

### Phong cách chơi

- Keeper chuyên khai thác Recovery và intraday reversal.
- Phù hợp với build Contrarian.
- Có thể tạo điểm rất cao trong ngày giảm sâu rồi hồi mạnh.

### Điểm mạnh

- Mạnh trong Recovery và Shock có phục hồi.
- Tương tác tốt với Strategy Counterflow.
- Có khả năng biến tình huống xấu thành điểm số.

### Điểm yếu

- Yếu trong Broad Growth ổn định.
- Passive không kích hoạt nếu không có đủ Depth.
- Rủi ro cao nếu giảm sâu nhưng không phục hồi.

### Tag đề xuất

```text
Contrarian
Recovery
Resurface
High Risk
```

### Hướng nâng cấp

#### Deeper Return

```text
Giảm yêu cầu phục hồi từ 50% xuống 40% từ đáy.
```

#### Hidden Exit

```text
Nếu không đạt điều kiện Resurface,
giảm một phần Depth Penalty thay vì không nhận hiệu ứng.
```

---

## 3.7. Bloom Creditor

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Bloom |
| Role | Compounder |
| Rarity | Rare |
| Base Risk | Trung bình |
| Nhóm tham chiếu | Lending và Borrowing |
| Current hiển thị | Lending Current |

### Basket tham chiếu

```text
AAVE 30%
MORPHO 25%
SYRUP 15%
COMP 15%
KMNO 15%
```

### Fantasy

Bloom Creditor là một người làm vườn san hô có thể biến những giọt nước nhỏ thành cả khu rừng. Nhân vật đại diện cho sức mạnh tăng dần theo thời gian, tích lũy giá trị khi được giữ ổn định trong Fleet.

### Passive — Accrued Tide

```text
Mỗi ngày liên tiếp được giữ trong Fleet:
+4% Skill Effect.

Tối đa 5 ngày:
+20% Skill Effect.
```

### Hiệu ứng Objective

```text
Nếu hoàn thành Daily Objective,
có khả năng nhận thêm 1 Supplies.
```

Việc xác định phần thưởng phải sử dụng deterministic seed của Voyage, không dùng random không kiểm soát.

### Phong cách chơi

- Compounder dành cho build dài hạn.
- Thưởng cho người chơi giữ đội hình ổn định.
- Có giá trị kinh tế và progression tốt.

### Điểm mạnh

- Scaling tốt qua nhiều ngày.
- Tạo thêm lợi ích khi hoàn thành Objective.
- Phù hợp với Calm Waters và Voyage dài.

### Điểm yếu

- Không mạnh ngay khi mới tuyển.
- Bị mất nhịp nếu phải thay khỏi Fleet.
- Chịu thêm Risk Penalty trong Shock do áp lực thanh lý.

### Tag đề xuất

```text
Lending
Economy
Compounder
Long-term
```

### Hướng nâng cấp

#### Deep Roots

```text
Giữ lại một nửa stack Accrued Tide
khi Keeper bị đưa khỏi Fleet trong một ngày.
```

#### Shared Yield

```text
Khi đạt tối đa stack,
chia sẻ một phần Skill Effect cho Keeper bên cạnh.
```

---

## 3.8. Ember Trickster

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Ember |
| Role | Trickster |
| Rarity | Common |
| Base Risk | Rất cao |
| Nhóm tham chiếu | Meme Coin |
| Current hiển thị | Wild Current |

### Basket tham chiếu

```text
DOGE 30%
SHIB 20%
PEPE 20%
PUMP 15%
BONK 15%
```

### Fantasy

Ember Trickster là một con cáo lửa cưỡi tên lửa tự chế và luôn cười ngay cả khi đang rơi. Nhân vật đại diện cho momentum mạnh, biến động lớn và khả năng tạo kết quả cực đoan.

### Passive — Wild Surge

```text
Nếu Normalized Performance > +1.0:
nhân 1.5 lần Sector Relative Score.

Nếu Normalized Performance < -1.0:
nhận thêm 30% Pressure Penalty.
```

### Phong cách chơi

- Keeper risk/reward rõ ràng nhất trong roster.
- Có thể tạo điểm rất cao trong ngày thuận lợi.
- Có thể gây Hull Damage lớn nếu dự đoán sai hoặc gặp Broad Decline.

### Điểm mạnh

- Performance Ceiling cao.
- Mạnh trong Momentum và Wild Current.
- Tạo lựa chọn chiến thuật hấp dẫn cho người chơi thích mạo hiểm.

### Điểm yếu

- Kết quả biến động mạnh.
- Cần Warden hoặc Hedge để kiểm soát rủi ro.
- Dễ bị phạt trong Shock hoặc Broad Decline.

### Tag đề xuất

```text
Trickster
Momentum
High Variance
Risk Reward
```

### Hướng nâng cấp

#### Burning Tail

```text
Tăng multiplier Sector Relative Score từ 1.5 lên 1.75.
```

#### Last Laugh

```text
Giới hạn Pressure Penalty tối đa một lần mỗi Tide.
```

---

## 3.9. Forge Architect

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Forge |
| Role | Growth |
| Rarity | Epic |
| Base Risk | Cao |
| Nhóm tham chiếu | DePIN, compute và dữ liệu |
| Current hiển thị | Engine Current |

### Basket tham chiếu

```text
TAO 30%
RENDER 25%
FIL 20%
GRASS 15%
GRT 10%
```

### Fantasy

Forge Architect là một thợ rèn bạch tuộc xây dựng các cỗ máy dưới biển. Mỗi xúc tu vận hành một node riêng, nhưng toàn bộ hệ thống chỉ phát huy sức mạnh khi nhiều bộ phận cùng hoạt động.

### Passive — Distributed Engine

```text
Nếu đồng thời thỏa mãn:

- Basket return dương.
- Volume Trend dương.
- Ít nhất 3 trong 5 thành phần tăng.

Nhận +30% Performance Ceiling.
```

### Phong cách chơi

- Growth Keeper yêu cầu breadth thật sự.
- Không thưởng cho trường hợp chỉ một thành phần kéo toàn basket.
- Phù hợp với người chơi muốn kiểm tra độ rộng và chất lượng của xu hướng.

### Điểm mạnh

- Điểm bùng nổ cao khi cả nhóm cùng tăng.
- Tốt trong Broad Growth.
- Có thể trở thành carry chính của Forge build.

### Điểm yếu

- Không kích hoạt nếu breadth thấp.
- Có thể mất nhiều điểm khi narrative đảo chiều.
- Rarity cao và nên khó tuyển hơn các Keeper khác.

### Tag đề xuất

```text
Growth
Infrastructure
Breadth
Performance Ceiling
```

### Hướng nâng cấp

#### More Nodes

```text
Giảm điều kiện breadth từ 3/5 xuống 3/6
và thêm một thành phần nhỏ vào basket version mới.
```

#### Redundant Engine

```text
Nếu một thành phần giảm mạnh,
giảm ảnh hưởng của thành phần đó lên Depth Penalty.
```

---

## 3.10. Crest Pathfinder

### Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Sector | Crest |
| Role | Growth |
| Rarity | Rare |
| Base Risk | Trung bình–cao |
| Nhóm tham chiếu | Layer 2 |
| Current hiển thị | Second Current |

### Basket tham chiếu

```text
MNT 40%
POL 35%
ARB 25%
```

### Fantasy

Crest Pathfinder là một chim hải âu cơ khí bay trên tầng sóng thứ hai, tìm ra các lối đi nhanh hơn bên trên những Current lớn. Nhân vật mạnh khi Layer 2 tăng nhanh hơn nền tảng Layer 1 bên dưới.

### Passive — Second Current

```text
Nếu Layer 1 benchmark dương
và Second Current vượt Layer 1 benchmark:

chuyển 25% Sector Relative Score
thành bonus cho toàn Fleet.
```

### Risk

```text
Nếu Layer 1 benchmark giảm sâu:
nhận thêm 20% Depth Penalty.
```

### Phong cách chơi

- Growth Keeper phụ thuộc vào môi trường Layer 1.
- Mạnh khi Layer 2 thể hiện relative strength.
- Tạo bonus cho cả Fleet nếu vượt được nền tảng tham chiếu.

### Điểm mạnh

- Có khả năng tạo Fleet-wide bonus.
- Hoạt động tốt trong Broad Growth hoặc Layer 2 rotation.
- Tương tác tốt với Crest Sovereign.

### Điểm yếu

- Nhạy cảm với Layer 1 crash.
- Basket có ít thành phần nên cần kiểm soát concentration.
- Có thể yếu nếu Layer 1 tăng quá mạnh còn Layer 2 đi ngang.

### Tag đề xuất

```text
Layer 2
Growth
Relative Strength
Fleet Bonus
```

### Hướng nâng cấp

#### Higher Path

```text
Tăng lượng Sector Relative Score chuyển cho Fleet
 từ 25% lên 35%.
```

#### Safety Route

```text
Giảm Depth Penalty bổ sung trong Layer 1 decline
 từ 20% xuống 10%.
```

---

# 4. Bảng tổng hợp roster

| Keeper | Sector | Role | Rarity | Kiểu chơi chính | Mạnh trong | Yếu trong |
|---|---|---|---|---|---|---|
| Crest Sovereign | Crest | Vanguard | Common | Cân bằng | Broad Growth, Broad Decline | Rotation |
| Harbor Warden | Harbor | Warden | Common | Phòng thủ | Shock, Broad Decline | Strong Growth |
| Iron Broker | Forge | Compounder | Rare | Tích lũy | Nhiều ngày ổn định | Liquidity Shock |
| Current Weaver | Current | Support | Common | Synergy đa Sector | Rotation | Calm, volume thấp |
| Echo Seer | Echo | Oracle | Rare | Thông tin | Signal phức tạp | Raw performance race |
| Veil Reversalist | Veil | Contrarian | Rare | Phục hồi | Recovery, Shock | Smooth Growth |
| Bloom Creditor | Bloom | Compounder | Rare | Tích lũy dài hạn | Calm, Sideways | Shock |
| Ember Trickster | Ember | Trickster | Common | Risk/reward | Momentum, Wild Current | Broad Decline |
| Forge Architect | Forge | Growth | Epic | Tăng trưởng hạ tầng | Broad Growth | Narrative reversal |
| Crest Pathfinder | Crest | Growth | Rare | Relative strength | Layer 1 Growth | Layer 1 crash |

---

# 5. Đội hình khởi đầu đề xuất

Ba Keeper khởi đầu phù hợp cho người chơi mới:

```text
Crest Sovereign
Harbor Warden
Current Weaver
```

Đội hình này cung cấp:

- Một Vanguard giữ tuyến đầu.
- Một Warden kiểm soát Depth.
- Một Support tạo Diversification.
- Ba kiểu basket khác nhau.
- Khả năng hoạt động trong nhiều Market Regime.
- Ít phụ thuộc vào một kết quả cực đoan.

Ember Trickster nên xuất hiện sớm trong shop để giới thiệu cơ chế risk/reward. Echo Seer, Forge Architect và các Keeper scaling nên xuất hiện sau khi người chơi đã hiểu Signal, benchmark và settlement.

---

# 6. Gợi ý phân bố rarity

```text
Common
- Crest Sovereign
- Harbor Warden
- Current Weaver
- Ember Trickster

Rare
- Iron Broker
- Echo Seer
- Veil Reversalist
- Bloom Creditor
- Crest Pathfinder

Epic
- Forge Architect
```

Rarity không nên chỉ phản ánh sức mạnh. Nó nên phản ánh:

- Độ phức tạp của mechanic.
- Mức độ phụ thuộc vào build.
- Khả năng thay đổi cách chơi của Fleet.
- Tần suất xuất hiện phù hợp trong shop.

---

# 7. Nguyên tắc triển khai gameplay

## 7.1. Keeper Market Performance

Mỗi Keeper lấy dữ liệu từ basket tương ứng:

```text
Basket Return
= Σ(Component Weight × Component Return)
```

Sau đó chuẩn hóa:

```text
Normalized Performance
= Basket Return / Expected Turbulence
```

Giới hạn đề xuất:

```text
Normalized Performance ∈ [-2.0, +2.0]
```

## 7.2. Keeper Market Score

```text
Keeper Market Score
= 40% Global Relative Score
+ 40% Sector Relative Score
+ 20% Risk-adjusted Score
```

## 7.3. Fleet Performance

Công thức MVP dễ giải thích:

```text
Base Fleet Performance
= trung bình Keeper Market Score trong các slot đang hoạt động
```

Sau đó áp dụng:

```text
Adjusted Fleet Performance
= Base Fleet Performance
+ Keeper Passive
+ Upgrade
+ Synergy
+ Relic
+ Strategy
+ Daily Modifier
```

## 7.4. Daily Score

```text
Daily Score
= 40% Relative Performance
+ 20% Sector Ranking
+ 15% Depth Control
+ 10% Resurface
+ 10% Synergy & Skill Effects
+ 5% Daily Objective
```

## 7.5. Phân tách dữ liệu và gameplay

Raw market data không được chỉnh sửa bởi Passive.

Passive chỉ nên tác động lên:

- Score component.
- Penalty.
- Cap.
- Multiplier.
- Hull Damage.
- Supplies reward.
- Signal visibility.

Điều này giúp settlement dễ kiểm tra, replay và giải thích cho người chơi.

---

# 8. Các vấn đề cần kiểm chứng qua playtest

- Harbor Warden có khiến build phòng thủ quá an toàn không?
- Ember Trickster có tạo variance quá lớn không?
- Echo Seer có đủ giá trị dù Market Score thấp không?
- Iron Broker và Bloom Creditor có scaling quá mạnh trong ngày cuối không?
- Forge Architect có quá khó kích hoạt so với rarity không?
- Crest Sovereign có trở thành lựa chọn mặc định trong mọi Fleet không?
- Crest Pathfinder có quá tương quan với Crest Sovereign không?
- Stablecoin basket nên dùng Peg Stability hay thêm Yield/Liquidity metric?
- Basket ba thành phần có cần concentration penalty riêng không?
- Các Compounder có khiến người chơi ngại thay đổi đội hình không?

---

# 9. Kết luận

Roster 10 Keeper này cung cấp đầy đủ các archetype cần thiết cho MVP:

```text
Vanguard
Warden
Growth
Support
Oracle
Trickster
Contrarian
Compounder
```

Các Keeper tạo ra lựa chọn giữa:

- An toàn và tăng trưởng.
- Thông tin và raw score.
- Cam kết dài hạn và linh hoạt ngắn hạn.
- Đội hình tập trung và đa dạng hóa.
- Momentum và Recovery.
- Risk và Reward.

Nguyên tắc cốt lõi của roster:

> Coin tạo dữ liệu cho Current. Current tạo Market Performance cho Keeper. Keeper tạo chiến thuật cho Fleet.
