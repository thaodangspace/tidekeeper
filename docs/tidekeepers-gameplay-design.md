# Tidekeepers

## Gameplay Design Document

**Thể loại:** Daily asynchronous strategy roguelite  
**Nền tảng:** Web  
**Thời lượng:** 3–5 phút mỗi ngày  
**Độ dài một hành trình:** 7–14 ngày  
**Chế độ chính:** Chơi đơn theo mùa, có thể mở rộng sang PvP bất đồng bộ  
**Nguồn biến động:** Dữ liệu thị trường công khai ngoài đời thực  
**Tiền thật:** Không sử dụng  
**AI:** Không cần trong MVP  

---

## 1. Tóm tắt ý tưởng

**Tidekeepers** là một game chiến thuật bất đồng bộ theo ngày. Người chơi xây dựng một đội hình gồm các nhân vật đại diện cho nhiều phong cách tăng trưởng, rủi ro và phòng thủ khác nhau.

Mỗi ngày, đội hình được khóa và bước vào một đợt **Tide** mới. Biến động dữ liệu thị trường thật trong 24 giờ tạo ra điều kiện chiến đấu chung cho toàn bộ người chơi. Kết quả không được quyết định đơn thuần bởi tài sản tăng hay giảm, mà bởi khả năng:

- Vượt hiệu suất thị trường chung.
- Chọn đúng đội hình cho trạng thái hiện tại.
- Kiểm soát rủi ro.
- Kích hoạt synergy.
- Tận dụng kỹ năng nhân vật.
- Sống sót qua các thử thách của từng ngày.
- Thích nghi trong suốt một hành trình nhiều ngày.

Mục tiêu của người chơi không phải là dự đoán chính xác giá trị ngày mai, mà là xây dựng một đội hình có khả năng ứng phó tốt với nhiều kịch bản khác nhau.

> **Build today. Face tomorrow.**

---

## 2. Fantasy của người chơi

Người chơi vào vai một **Tidekeeper** — người dẫn dắt một đoàn sinh vật hoặc chiến binh có khả năng khai thác, chống chịu và chuyển hóa các dòng biến động.

Thế giới Tidekeepers liên tục bị tác động bởi những đợt thủy triều vô hình:

- Growth Tide.
- Red Tide.
- Volatility Storm.
- Rotation Current.
- Recovery Wave.
- Black Tide.
- Calm Waters.

Các đợt Tide được tạo từ dữ liệu ngoài đời thật, nhưng được chuyển hóa thành ngôn ngữ fantasy để game không mang cảm giác của một ứng dụng giao dịch.

Người chơi không sở hữu tài sản thật. Họ chỉ sở hữu các **Keepers**, **Relics**, **Strategies** và **Blessings** trong game.

---

## 3. Trụ cột thiết kế

### 3.1. Quyết định ngắn, hệ quả dài

Mỗi ngày người chơi chỉ cần vài phút để:

1. Xem kết quả hôm trước.
2. Đọc các tín hiệu mới.
3. Chỉnh sửa đội hình.
4. Chọn chiến lược.
5. Khóa đội hình.

Kết quả được giải quyết sau đó bằng dữ liệu của cả ngày.

### 3.2. Random phải có thể đọc được

Người chơi không được biết chính xác tương lai, nhưng luôn phải nhận được tín hiệu hoặc dữ liệu đủ để đưa ra quyết định có cơ sở.

### 3.3. Thắng tương đối, không thắng tuyệt đối

Một ngày tất cả chỉ số đều giảm vẫn có thể là một ngày thắng nếu đội hình giảm ít hơn benchmark, kiểm soát drawdown tốt hoặc hoàn thành mục tiêu đặc biệt.

### 3.4. Build quan trọng hơn dự đoán

Dữ liệu thật tạo ra chiến trường. Đội hình, synergy, relic và strategy mới quyết định khả năng sống sót.

### 3.5. Kết quả phải giải thích được

Sau mỗi ngày, người chơi luôn thấy breakdown:

- Điều kiện thị trường.
- Hiệu suất từng Keeper.
- Kỹ năng đã kích hoạt.
- Bonus synergy.
- Penalty rủi ro.
- Benchmark.
- Damage hoặc phần thưởng cuối cùng.

---

## 4. Vòng lặp gameplay chính

```text
Nhận kết quả hôm qua
→ Nhận Capital và phần thưởng
→ Xem tín hiệu hôm nay
→ Mua, bán hoặc nâng cấp Keeper
→ Sắp xếp đội hình
→ Chọn Strategy
→ Khóa đội hình
→ Dữ liệu thị trường chạy trong 24 giờ
→ Hệ thống giải quyết kết quả
→ Ngày hôm sau quay lại
```

Mỗi lần quay lại phải tạo được ba cảm giác:

1. **Tò mò:** Hôm qua đội hình của mình đã hoạt động thế nào?
2. **Hiểu:** Vì sao mình thắng hoặc thua?
3. **Chủ động:** Hôm nay mình sẽ thay đổi gì?

---

## 5. Cấu trúc một hành trình

Một hành trình, hay **Voyage**, kéo dài từ 7 đến 14 ngày.

### Ví dụ Voyage 14 ngày

| Ngày | Loại thử thách |
|---|---|
| 1 | Normal Tide |
| 2 | Normal Tide |
| 3 | Special Objective |
| 4 | Mini Boss |
| 5 | Route Choice |
| 6 | Normal Tide |
| 7 | Elite Tide |
| 8 | Mini Boss |
| 9 | Special Objective |
| 10 | Elite Tide |
| 11 | Route Choice |
| 12 | High Volatility Tide |
| 13 | Preparation for Final Tide |
| 14 | Final Boss |

### Điều kiện kết thúc

Người chơi kết thúc Voyage khi:

- Fund Health về 0.
- Thất bại trước Final Boss.
- Hoàn thành toàn bộ hành trình.
- Chủ động từ bỏ Voyage.

### Điểm tổng kết

Điểm cuối hành trình có thể dựa trên:

```text
Voyage Score
= Performance Score
+ Remaining Fund Health
+ Relic Difficulty Bonus
+ Boss Completion Bonus
+ Optional Challenge Bonus
```

---

## 6. Lịch game theo ngày

Hệ thống nên sử dụng một múi giờ cố định cho toàn server.

Ví dụ:

```text
06:55 ICT  Khóa đội hình
07:00 ICT  Bắt đầu Tide mới
07:00 ICT ngày tiếp theo  Đóng dữ liệu
07:00–07:05  Xử lý kết quả
07:05  Công bố kết quả và mở preparation mới
```

Người chơi có thể chỉnh sửa đội hình bất cứ lúc nào trong thời gian preparation. Hệ thống chỉ sử dụng snapshot tại thời điểm khóa.

### Quy tắc quan trọng

- Không cho thay đổi đội hình sau thời điểm khóa.
- Mỗi ngày chỉ có một snapshot hợp lệ.
- Kết quả phải dựa trên cùng một cửa sổ thời gian cho mọi người.
- Dữ liệu dùng để settle phải được lưu lại.
- Không thay thế dữ liệu thật bằng random nếu API lỗi.
- Mọi kết quả phải gắn với phiên bản game rule.

---

## 7. Dữ liệu thị trường thật

### 7.1. Vai trò của dữ liệu thật

Dữ liệu thật chỉ tạo ra:

- Xu hướng chung.
- Hiệu suất tương đối.
- Volatility.
- Drawdown.
- Recovery.
- Tương quan giữa các nhóm.
- Market regime của ngày.

Dữ liệu thật không trực tiếp quyết định người thắng.

### 7.2. Không dùng giá tuyệt đối

Game chỉ dùng phần trăm biến động:

```text
Return = (Close Price - Open Price) / Open Price
```

Không dùng giá của tài sản làm sức mạnh cơ bản.

### 7.3. Không map nhân vật trực tiếp vào một tài sản duy nhất trong MVP

Mỗi Keeper nên đại diện cho một **basket**, một phong cách hoặc một nhóm.

Ví dụ:

```text
Crest Guardian
= weighted basket của nhóm vốn hóa lớn

Ember Trickster
= weighted basket của nhóm biến động cao

Current Weaver
= weighted basket của nhóm hạ tầng

Harbor Warden
= nhóm tài sản ổn định hoặc logic phòng thủ
```

Lợi ích:

- Giảm tác động của một tài sản tăng hoặc giảm bất thường.
- Dễ cân bằng.
- Không phụ thuộc quá nhiều vào thương hiệu bên ngoài.
- Không buộc game phải dùng tên hay logo thật.
- Dễ thay đổi thành phần dữ liệu mà không đổi nhân vật.

### 7.4. Source of truth

MVP chỉ nên chọn một nhà cung cấp dữ liệu chính.

Hệ thống phải lưu:

- Provider.
- Asset identifier.
- Open timestamp.
- Close timestamp.
- Open price.
- Close price.
- Raw return.
- Volume hoặc volatility nếu sử dụng.
- Raw response hash.
- Settlement timestamp.

---

## 8. Chuẩn hóa dữ liệu

Nếu chỉ dùng raw return, nhóm biến động cao sẽ có lợi thế quá lớn.

Dữ liệu phải được chuẩn hóa theo mức biến động kỳ vọng:

```text
Normalized Performance
= Raw Return / Expected Volatility
```

Sau đó giới hạn:

```text
Normalized Performance ∈ [-2, +2]
```

### Ví dụ

| Basket | Raw return | Expected volatility | Normalized |
|---|---:|---:|---:|
| Crest | +4% | 4% | +1.00 |
| Ember | +12% | 15% | +0.80 |
| Harbor | +0.2% | 0.5% | +0.40 |

Nhóm Ember tăng mạnh hơn nhưng không tự động nhận điểm cao nhất, vì mức biến động bình thường của nó vốn đã lớn.

---

## 9. Benchmark và hiệu suất tương đối

Mỗi ngày hệ thống tính một benchmark chung:

```text
Global Benchmark
= weighted average return của toàn bộ basket trong game
```

Điểm tương đối:

```text
Relative Performance
= Portfolio Performance - Global Benchmark
```

### Ví dụ thị trường giảm

```text
Global Benchmark: -10%
Portfolio A:       -4%
Relative Score:    +6%
```

Portfolio A vẫn được xem là chơi tốt vì bảo toàn tốt hơn toàn thị trường.

### Ví dụ thị trường tăng

```text
Global Benchmark: +12%
Portfolio A:        +8%
Relative Score:     -4%
```

Portfolio A tăng nhưng vẫn chơi kém hơn thị trường.

---

## 10. Ba lớp tham chiếu

Để tránh việc một loại Keeper luôn vượt trội, mỗi nhân vật nên được chấm qua ba lớp:

```text
Keeper Market Score
= 40% Global Relative Score
+ 40% Sector Relative Score
+ 20% Risk-adjusted Score
```

### Global Relative Score

So với toàn bộ thị trường.

### Sector Relative Score

So với các basket có đặc tính tương tự.

### Risk-adjusted Score

So với mức volatility hoặc drawdown kỳ vọng.

---

## 11. Market Regime

Sau khi dữ liệu ngày hoàn tất, hệ thống phân loại ngày đó thành một regime.

### Broad Growth

- Phần lớn basket tăng.
- Tương quan dương cao.
- Momentum Keeper mạnh.
- Build quá phòng thủ khó vượt benchmark.

### Broad Decline

- Phần lớn basket giảm.
- Defender, Harbor và Contrarian mạnh.
- Leverage và Trickster gặp nguy hiểm.

### Rotation

- Một số nhóm tăng, nhóm khác giảm.
- Sector selection trở nên quan trọng.
- Diversification không phải lúc nào cũng tốt.

### Sideways

- Open và close không chênh lệch nhiều.
- Intraday volatility có thể cao.
- Keeper dựa vào range, shield hoặc stability có giá trị.

### Recovery

- Giá giảm sâu rồi phục hồi.
- Recovery skill và contrarian build mạnh.

### Shock

- Drawdown lớn.
- Correlation tăng mạnh.
- Các cơ chế bảo hiểm và circuit breaker được thử thách.

---

## 12. Đội hình

Người chơi bắt đầu với 3 slot và có thể mở rộng lên 5 slot trong một Voyage.

Mỗi slot chứa một Keeper.

```text
[ Keeper 1 ] [ Keeper 2 ] [ Keeper 3 ] [ Keeper 4 ] [ Keeper 5 ]
```

MVP không cần bàn cờ hoặc positioning phức tạp. Có thể thêm adjacency sau này.

### Quy tắc đội hình

- Một Keeper có thể xuất hiện nhiều bản sao nếu thiết kế nâng cấp yêu cầu.
- Mỗi Keeper có Sector, Role và Passive.
- Một số synergy yêu cầu nhiều Keeper cùng Sector.
- Một số synergy thưởng cho đa dạng hóa.
- Một số Relic thay đổi giới hạn slot hoặc trọng số.

---

## 13. Keeper

Mỗi Keeper có các thuộc tính:

```text
Name
Rarity
Sector
Role
Base Risk
Expected Volatility
Passive Skill
Upgrade Path
Tags
Basket Mapping
```

### Sector

Ví dụ:

- Crest.
- Ember.
- Current.
- Harbor.
- Veil.
- Forge.
- Bloom.
- Echo.

Sector quyết định Keeper phản ứng với loại Tide nào.

### Role

#### Vanguard

Chịu rủi ro và bảo vệ đồng đội.

#### Growth

Tạo Performance Score cao khi điều kiện thuận lợi.

#### Support

Buff Keeper khác hoặc tăng synergy.

#### Oracle

Cung cấp thêm dữ liệu và giảm độ mơ hồ của tín hiệu.

#### Trickster

Biến động cao, có thể tạo kết quả lớn.

#### Contrarian

Mạnh khi thị trường giảm hoặc phục hồi.

#### Compounder

Tăng sức mạnh theo số ngày giữ trong đội hình.

#### Warden

Giảm drawdown, damage hoặc volatility.

---

## 14. Keeper mẫu

### Crest Guardian

**Sector:** Crest  
**Role:** Vanguard  

**Passive — Anchor the Line**

```text
Nếu Global Benchmark giảm trên 5%,
giảm 25% Drawdown Penalty của toàn đội.
```

**Upgrade A — Deep Anchor**

```text
Tăng hiệu quả phòng thủ lên 35%.
```

**Upgrade B — Rising Crest**

```text
Khi Crest sector vượt benchmark,
nhận thêm Performance Score.
```

---

### Ember Trickster

**Sector:** Ember  
**Role:** Trickster  

**Passive — Wild Surge**

```text
Nếu Ember basket tăng trên ngưỡng volatility kỳ vọng,
nhân 1.5 lần Sector Relative Score.
```

**Rủi ro**

```text
Nếu Ember basket giảm dưới ngưỡng crash,
nhận thêm Risk Penalty.
```

---

### Harbor Warden

**Sector:** Harbor  
**Role:** Warden  

**Passive — Safe Passage**

```text
Mỗi ngày bảo vệ Keeper có Drawdown lớn nhất,
giảm 40% Drawdown Penalty của Keeper đó.
```

---

### Current Weaver

**Sector:** Current  
**Role:** Support  

**Passive — Flow Transfer**

```text
Nếu hai Sector khác nhau cùng vượt benchmark,
tăng Diversification Score.
```

---

### Echo Seer

**Sector:** Echo  
**Role:** Oracle  

**Passive — Read the Wake**

```text
Hiển thị thêm một tín hiệu định lượng:
momentum, volatility, correlation hoặc volume trend.
```

Echo Seer không dự đoán chắc chắn tương lai. Nhân vật chỉ cung cấp thêm thông tin.

---

### Dusk Reversalist

**Sector:** Veil  
**Role:** Contrarian  

**Passive — From the Depths**

```text
Nếu basket giảm sâu nhưng phục hồi mạnh từ đáy,
nhận thêm Recovery Score.
```

---

## 15. Nâng cấp Keeper

MVP có thể chọn một trong hai mô hình.

### Mô hình bản sao

```text
3 Keeper 1★ → Keeper 2★
3 Keeper 2★ → Keeper 3★
```

Ưu điểm:

- Dễ hiểu.
- Tạo động lực reroll.
- Gần với auto-battler.

Nhược điểm:

- Dễ tạo cảm giác sao chép TFT.
- Shop balance phức tạp hơn.

### Mô hình cây nâng cấp

Mỗi Keeper có hai nhánh.

Ví dụ:

```text
Harbor Warden
├── Deep Shelter
│   Tăng phòng thủ
└── Safe Yield
    Tăng điểm trong ngày ổn định
```

Ưu điểm:

- Hợp roguelite hơn.
- Mỗi Voyage có build khác nhau.
- Giảm phụ thuộc vào duplicate.

### Đề xuất MVP

Dùng **cây nâng cấp hai nhánh**. Duplicate có thể được thêm trong phiên bản sau.

---

## 16. Synergy

Synergy tạo lý do để xây đội hình thay vì chọn các Keeper có raw score cao nhất.

### Crest

```text
2 Crest:
Giảm volatility toàn đội.

4 Crest:
Sau một ngày Broad Decline,
nhận Recovery Blessing ở ngày tiếp theo.
```

### Ember

```text
2 Ember:
Tăng Performance Ceiling và Risk.

4 Ember:
Khi một Ember Keeper vượt sector benchmark,
các Ember Keeper khác nhận Momentum.
```

### Harbor

```text
2 Harbor:
Giảm Fund Health damage.

3 Harbor:
Hồi một lượng nhỏ Fund Health nếu hoàn thành mục tiêu ngày.
```

### Echo

```text
2 Echo:
Hiển thị thêm một signal.

4 Echo:
Xác định signal nào có độ tin cậy thấp nhất.
```

### Diversified Fleet

```text
Nếu đội hình có 4 Sector khác nhau:
giảm Sector Concentration Penalty.
```

### Balanced Current

```text
Nếu đội hình có ít nhất:
1 Vanguard
1 Growth
1 Support
1 Warden

nhận bonus ổn định.
```

---

## 17. Relic

Relic là modifier tồn tại đến hết Voyage.

### Diamond Hold

```text
Keeper giữ trong đội hình ít nhất 3 ngày
nhận thêm Stability Score.
```

### Circuit Bell

```text
Mỗi ngày, lần đầu portfolio vượt ngưỡng drawdown,
giảm một phần damage.
```

### Flow Compass

```text
Hiển thị Sector Rotation mạnh nhất trước khi khóa đội hình.
```

### Balanced Ledger

```text
Mỗi Sector khác nhau tăng một lượng nhỏ Performance Score,
nhưng synergy đơn ngành bị giảm.
```

### Red Sail

```text
Nhân Risk và Reward của toàn đội.
```

### Quiet Harbor

```text
Giảm Capital nhận được,
đổi lại hồi Fund Health thường xuyên hơn.
```

---

## 18. Strategy

Strategy là lựa chọn ngắn hạn, chỉ áp dụng trong một ngày.

### Hedge

```text
Giảm một nửa cả Performance Gain và Risk Penalty.
```

### Lock Gains

```text
Giảm Recovery bonus,
nhưng bảo vệ một phần điểm đã đạt được.
```

### Rotate

```text
Chuyển một phần Sector bonus từ Keeper này sang Keeper khác.
```

### Take the Current

```text
Tăng Momentum Score,
nhưng giảm khả năng chống đảo chiều.
```

### Counterflow

```text
Nhận bonus nếu Global Benchmark giảm,
nhưng bị penalty trong Broad Growth.
```

Người chơi chọn Strategy trước khi khóa đội hình. MVP không cần thao tác realtime trong lúc Tide diễn ra.

---

## 19. Signal

Signal là dữ liệu được biến đổi thành thông tin dễ đọc.

Ví dụ:

```text
“Dòng chảy chung đang mạnh lên.”
“Biến động của Ember sector đang cao hơn bình thường.”
“Các nhóm phòng thủ đang ít tương quan với phần còn lại.”
“Volume của Current sector đang tăng.”
```

### Các loại Signal

- Momentum.
- Volatility.
- Volume trend.
- Correlation.
- Sector breadth.
- Drawdown.
- Recovery strength.
- Market concentration.

### Độ chính xác

Signal không được nói chắc chắn:

```text
Sai:
“Ember sẽ tăng ngày mai.”
```

Nên nói:

```text
Đúng:
“Ember đang có momentum và volatility cao.”
```

### Signal noise

Sau MVP có thể thêm signal có độ tin cậy khác nhau. Oracle Keeper giúp nhận diện noise.

---

## 20. Daily Modifier

Mỗi ngày có một luật đặc biệt được công bố trước khi khóa đội hình.

### Flight to Harbor

```text
Warden và Vanguard nhận thêm defensive effect.
Trickster nhận thêm Risk.
```

### Wild Current

```text
Keeper có |return| cao nhận bonus,
nhưng Drawdown Penalty cũng tăng.
```

### Rotation Day

```text
Sector Relative Score có trọng số cao hơn Global Score.
```

### Calm Waters

```text
Volatility thấp.
Compounder và Stability build nhận bonus.
```

### Recovery Tide

```text
Recovery Score được nhân đôi.
```

### Concentration Audit

```text
Đội hình có quá nhiều Keeper cùng Sector nhận penalty.
```

Daily Modifier giúp ngày Broad Growth hoặc Broad Decline vẫn tạo ra trải nghiệm khác nhau.

---

## 21. Mục tiêu ngày

Không phải ngày nào cũng yêu cầu đạt return cao nhất.

### Outperform

Vượt Global Benchmark.

### Preserve

Giữ Max Drawdown dưới ngưỡng.

### Recover

Đạt Recovery Score tối thiểu.

### Diversify

Có đủ số Sector hoạt động tích cực.

### Control Risk

Đạt Performance Score mà không vượt Risk Budget.

### Sector Trial

Một Sector cụ thể có trọng số cao hơn.

### Survival

Không nằm trong nhóm kết quả thấp nhất.

---

## 22. Công thức điểm

Công thức MVP đề xuất:

```text
Daily Score
= 40% Relative Performance
+ 20% Sector Ranking
+ 15% Drawdown Control
+ 10% Recovery
+ 10% Synergy & Skill Effects
+ 5% Daily Objective
```

### Relative Performance

Hiệu suất portfolio so với Global Benchmark.

### Sector Ranking

Hiệu suất các Keeper so với nhóm tương đồng.

### Drawdown Control

Khả năng tránh hoặc giảm mức sụt giảm lớn nhất.

### Recovery

Khả năng hồi phục từ đáy.

### Synergy & Skill Effects

Điểm từ Keeper, Relic và Strategy.

### Daily Objective

Điểm thưởng từ mục tiêu đặc biệt.

---

## 23. Fund Health

Fund Health là thanh sinh tồn của Voyage.

Người chơi bắt đầu với:

```text
100 Fund Health
```

### Damage

```text
Nếu Daily Score dưới ngưỡng:
Damage = khoảng cách tới ngưỡng × difficulty multiplier
```

### Hồi phục

Fund Health có thể hồi bằng:

- Harbor synergy.
- Recovery node.
- Relic.
- Boss reward.
- Hoàn thành objective đặc biệt.

### Vì sao tách Fund Health khỏi Capital

Nếu dùng cùng một tài nguyên cho máu và mua sắm, người dẫn trước dễ snowball.

Nên tách:

```text
Fund Health: khả năng sống sót
Capital: tiền mua sắm
Voyage Score: thành tích
```

---

## 24. Capital và shop

Mỗi ngày người chơi nhận Capital cơ bản.

```text
Base Capital
+ Performance Bonus
+ Economy Effect
+ Event Reward
```

Capital dùng để:

- Tuyển Keeper.
- Mua nâng cấp.
- Mua Relic.
- Reroll shop.
- Mở thêm slot.
- Mua Strategy.
- Hồi Fund Health trong một số node.

### Ví dụ chi phí MVP

```text
Common Keeper: 3 Capital
Rare Keeper: 5 Capital
Reroll: 1 Capital
Upgrade: 4–6 Capital
Relic: 6–8 Capital
New slot: 8 Capital
```

---

## 25. Map roguelite

Bản MVP có thể chạy tuyến tính. Sau đó bổ sung bản đồ phân nhánh.

### Các loại node

#### Normal Tide

Một ngày bình thường.

#### Elite Tide

Luật khó hơn, thưởng Relic hiếm.

#### Harbor

Hồi Fund Health hoặc giảm Risk.

#### Market

Shop đặc biệt.

#### Oracle Tower

Xem thêm signal hoặc preview modifier.

#### Event

Lựa chọn có đánh đổi.

#### Mini Boss

Thử thách có luật riêng.

#### Final Tide

Boss cuối Voyage.

---

## 26. Event lựa chọn

Ví dụ:

> Một đoàn tàu lạ đề nghị chia sẻ dữ liệu độc quyền.

### Chấp nhận

```text
Nhận một Signal hiếm.
Tăng Risk trong hai ngày.
```

### Từ chối

```text
Nhận Fund Health.
```

### Điều tra

```text
Có khả năng nhận Relic hiếm,
nhưng cũng có thể mất Capital.
```

Event tạo thêm quyết định ngoài việc tối ưu chỉ số.

---

## 27. Boss

Boss không phải là một đối thủ có giá riêng. Boss là một bộ luật ép người chơi thay đổi build.

### The Red Leviathan

```text
Mỗi lần Global Benchmark giảm,
Keeper có Risk cao nhất nhận thêm pressure.
```

Counter:

- Warden.
- Diversification.
- Hedge.
- Circuit Bell.

### The Glass Current

```text
Trong ngày Broad Growth,
Performance tăng nhanh nhưng Drawdown Penalty bị nhân đôi nếu đảo chiều.
```

Counter:

- Lock Gains.
- Stable build.
- Keeper có take-profit effect.

### The Silent Deep

```text
Signal bị giảm chất lượng.
Oracle skill yếu đi.
```

Counter:

- Build cân bằng.
- Relic lưu trữ signal.
- Không phụ thuộc quá nhiều vào forecast.

### The Shifting Crown

```text
Sector dẫn đầu thay đổi theo từng khoảng trong ngày.
Sector Concentration bị phạt.
```

Counter:

- Diversification.
- Support.
- Rotation Strategy.

### The Final Tide

Boss cuối kết hợp:

- Một modifier công khai.
- Một rule ẩn có thể suy ra từ signal.
- Mục tiêu Performance.
- Giới hạn Drawdown.
- Yêu cầu sống sót.

---

## 28. Một ngày mẫu

### Trước khi khóa

Daily Modifier:

```text
Flight to Harbor
Warden nhận thêm defensive effect.
Trickster nhận thêm Risk.
```

Signal:

```text
Global momentum đang yếu.
Ember volatility đang tăng.
Crest sector giữ ổn định hơn phần còn lại.
```

Đội hình:

```text
Crest Guardian
Harbor Warden
Echo Seer
Current Weaver
Ember Trickster
```

Strategy:

```text
Hedge
```

### Dữ liệu thực sau 24 giờ

```text
Global Benchmark: -8%
Crest:             -4%
Harbor:            +0.1%
Current:           -9%
Echo:              -6%
Ember:            -15%
```

### Breakdown

```text
Raw Portfolio Performance        -6.4
Relative Performance             +1.6
Crest Sector Advantage           +1.2
Harbor Protection                +1.0
Hedge Risk Reduction             +0.8
Ember Risk Penalty               -0.6
Diversification Bonus            +0.5
Daily Objective                  +0.4

Final Daily Score                +4.9
Benchmark threshold              +2.0
Result                           WIN
```

Dù phần lớn dữ liệu giảm, người chơi vẫn thắng vì build phòng thủ tốt hơn thị trường chung.

---

## 29. Kết quả ngày

Màn hình kết quả nên có ba tầng.

### Tầng 1: Kết luận nhanh

```text
YOU SURVIVED THE RED TIDE
Fund Health: 82 → 86
Capital gained: 6
```

### Tầng 2: Diễn biến

Hiển thị timeline hoặc biểu đồ:

- Portfolio.
- Benchmark.
- Drawdown.
- Recovery.
- Skill triggers.

### Tầng 3: Breakdown

```text
Relative Performance
Sector Score
Risk Control
Recovery
Keeper Skills
Synergies
Relics
Daily Objective
```

Người chơi phải hiểu điều gì đã hoạt động và điều gì cần thay đổi.

---

## 30. Progression trong Voyage

Sau mỗi ngày, người chơi chọn một phần thưởng.

Ví dụ:

```text
A. Nhận một Keeper mới.
B. Nâng cấp một Keeper hiện có.
C. Nhận một Relic.
```

Hoặc:

```text
A. +2 Fund Health.
B. +3 Capital.
C. Nhìn trước Daily Modifier ngày mai.
```

Các lựa chọn phải tạo đánh đổi giữa:

- Sức mạnh hiện tại.
- Khả năng sống sót.
- Tiềm năng cuối Voyage.
- Thông tin.
- Tính linh hoạt.

---

## 31. Meta progression

Sau mỗi Voyage, người chơi mở khóa nội dung theo chiều ngang.

### Có thể mở khóa

- Keeper mới.
- Relic mới.
- Strategy mới.
- Tidekeeper class mới.
- Daily Modifier mới.
- Boss mới.
- Cosmetic.
- Difficulty.

### Không nên mở khóa

```text
Mọi Keeper vĩnh viễn +20% sức mạnh.
```

Điều này làm người chơi mới luôn yếu hơn.

---

## 32. Tidekeeper class

Mỗi người chơi chọn một nhân vật quản lý trước Voyage.

### The Warden

```text
Fund Health cao hơn.
Capital khởi đầu thấp hơn.
```

### The Seer

```text
Nhận thêm một Signal mỗi ngày.
Shop đắt hơn.
```

### The Drifter

```text
Reroll đầu tiên mỗi ngày miễn phí.
Đội hình khởi đầu yếu hơn.
```

### The Gambler

```text
Reward và Risk cao hơn.
```

### The Archivist

```text
Relic có hiệu quả tốt hơn,
nhưng giới hạn số Keeper.
```

MVP chỉ cần một Tidekeeper mặc định.

---

## 33. PvP bất đồng bộ

PvP không cần trong MVP, nhưng kiến trúc nên hỗ trợ từ đầu.

### Cách hoạt động

Hai người chơi:

- Có lineup snapshot riêng.
- Sử dụng cùng daily market data.
- Sử dụng cùng modifier.
- Sử dụng cùng rule version.

Hệ thống so sánh Daily Score.

### Chế độ có thể bổ sung

#### Daily Duel

Một trận 1v1 mỗi ngày.

#### League

8–20 người cạnh tranh trong một season.

#### Ghost Voyage

Người chơi đối đầu đội hình snapshot của người khác.

#### Guild Tide

Tổng điểm của nhiều thành viên.

---

## 34. Chống pay-to-win

Có thể thương mại hóa bằng:

- Skin Keeper.
- Board theme.
- Result animation.
- Profile frame.
- Voyage pass.
- Emote.
- Cosmetic Tide effects.
- Story chapter.
- Alternate voice pack.

Không bán:

- Dữ liệu dự báo chính xác hơn.
- Daily Score.
- Risk reduction trực tiếp.
- Keeper có chỉ số vượt trội.
- Extra settlement attempt.
- Quyền thay đội hình sau khi khóa.

---

## 35. Phạm vi MVP

### Nội dung

```text
1 Tidekeeper class
10 Keeper
4 Sector
4 Role
10 Relic
5 Strategy
7 Daily Modifier
8 Daily Objective
2 Mini Boss
1 Final Boss
1 Voyage 7 ngày
5 slot tối đa
```

### Hệ thống

```text
Đăng nhập
Tạo Voyage
Shop
Đội hình
Signal
Khóa snapshot
Daily settlement
Result breakdown
Fund Health
Capital
Reward selection
Voyage completion
Admin rule configuration
```

### Không cần trong MVP

```text
AI
PvP
Realtime multiplayer
Phaser
Bàn cờ
Biểu đồ nến
Giá trị tài sản thật
Blockchain
Trading realtime
Guild
Chat
Meta progression lớn
Animation phức tạp
```

---

## 36. MVP core loop

```text
Ngày 1:
Chọn Keeper khởi đầu
→ xem Signal
→ khóa đội hình

Ngày 2:
Xem kết quả
→ nhận Reward
→ mua hoặc nâng Keeper
→ chọn Strategy
→ khóa đội hình

Ngày 3–6:
Lặp lại với Modifier và Objective mới

Ngày 7:
Đối đầu Final Tide
→ nhận Voyage Score
→ kết thúc hành trình
```

---

## 37. Các chỉ số cần theo dõi khi test

### Retention

- Người chơi có quay lại ngày thứ hai không?
- Có hoàn thành Voyage 7 ngày không?
- Có bắt đầu Voyage mới không?

### Hiểu game

- Người chơi có hiểu vì sao thắng hoặc thua không?
- Có đọc breakdown không?
- Có thay đổi build dựa trên kết quả không?

### Đa dạng build

- Có một Keeper luôn được chọn không?
- Có một Sector luôn vượt trội không?
- Build phòng thủ có quá an toàn không?
- Build rủi ro có quá phụ thuộc may mắn không?

### Economy

- Capital có đủ để tạo lựa chọn không?
- Reroll có đáng sử dụng không?
- Người chơi có bị khóa vào build quá sớm không?

### Daily cadence

- 24 giờ có quá lâu không?
- Người chơi có nhớ quay lại không?
- Màn hình kết quả có tạo đủ cảm giác phần thưởng không?

---

## 38. Nguyên tắc cân bằng

### Không để raw return thống trị

Dữ liệu thật phải qua normalization và benchmark.

### Không để phòng thủ thắng mọi boss

Một số objective phải yêu cầu Performance hoặc Recovery.

### Không để volatility trở thành lựa chọn tối ưu duy nhất

Risk Penalty và expected volatility phải có trọng số đủ lớn.

### Không để Signal trở thành đáp án

Signal chỉ cung cấp thông tin, không đảm bảo kết quả.

### Không để một ngày bất thường phá cả Voyage

Daily Score và Fund Health damage cần có giới hạn.

### Không thay rule giữa một ngày đang chạy

Mỗi settlement sử dụng rule version đã khóa từ đầu ngày.

---

## 39. Backend settlement flow

```text
Lock time
→ tạo player lineup snapshot
→ lưu daily modifier
→ lưu rule version
→ bắt đầu data window

Close time
→ gọi market data provider
→ xác thực dữ liệu
→ lưu raw snapshot
→ tính basket return
→ tính benchmark
→ phân loại regime
→ tính từng Keeper
→ áp dụng skill
→ áp dụng synergy
→ áp dụng relic
→ áp dụng strategy
→ tính daily score
→ tính Fund Health
→ cấp Capital và reward
→ công bố kết quả
```

### Tính idempotent

Khóa duy nhất:

```text
voyage_id + day_number + player_id
```

Settlement chạy lại không được tạo kết quả khác hoặc cấp thưởng hai lần.

---

## 40. Các bảng dữ liệu cốt lõi

```text
users
voyages
voyage_players
daily_rounds
daily_modifiers
daily_objectives

keepers
keeper_versions
keeper_basket_mappings
sectors
roles
synergies

player_keeper_instances
player_lineup_snapshots
player_relics
player_strategies

market_assets
market_price_snapshots
market_basket_returns
market_regimes

daily_results
daily_score_breakdowns
reward_choices

game_rule_versions
provider_responses
settlement_jobs
```

---

## 41. Trường hợp dữ liệu lỗi

### Provider không phản hồi

- Retry theo backoff.
- Không settle bằng random.
- Giữ trạng thái pending.
- Thông báo trì hoãn trong game.

### Thiếu một asset trong basket

- Dùng phần còn lại nếu vượt tỷ lệ coverage tối thiểu.
- Nếu không đủ coverage, basket được đánh dấu neutral.
- Quy tắc phải công khai.

### Giá bất thường

- Kiểm tra outlier.
- So sánh với previous close.
- Giới hạn normalized score.
- Lưu dữ liệu gốc để audit.

### Asset bị delist

- Không thêm vào basket mới.
- Không thay mapping giữa một ngày đang chạy.
- Chuyển mapping vào phiên bản game tiếp theo.

---

## 42. Trải nghiệm giao diện

### Home

- Voyage hiện tại.
- Thời gian còn lại trước khi khóa.
- Fund Health.
- Daily modifier.
- CTA xem kết quả hoặc chỉnh đội hình.

### Preparation

- Keeper collection.
- Shop.
- Lineup.
- Synergy.
- Signal.
- Strategy.
- Lock button.

### Locked

- Đội hình snapshot.
- Countdown đến kết quả.
- Signal và rule đã khóa.
- Không cho thay đổi.

### Result

- Kết luận.
- Timeline.
- Breakdown.
- Reward choice.
- Gợi ý thay đổi cho ngày tiếp theo.

### Voyage Map

- Ngày hiện tại.
- Boss sắp tới.
- Route choice.
- Relic đang sở hữu.
- Tổng score.

---

## 43. Ngôn ngữ trong game

Nên tránh thuật ngữ tài chính quá trực tiếp.

| Thuật ngữ hệ thống | Ngôn ngữ game |
|---|---|
| Market | Tide |
| Portfolio | Fleet |
| Asset group | Current |
| Price movement | Flow |
| Volatility | Turbulence |
| Benchmark | World Tide |
| Drawdown | Depth |
| Recovery | Resurface |
| Capital | Supplies |
| Risk | Pressure |
| Round | Day hoặc Tide |
| Season | Voyage |

Có thể vẫn dùng thuật ngữ phổ thông trong tooltip nâng cao, nhưng giao diện chính nên giữ chất fantasy.

---

## 44. Định vị sản phẩm

### Elevator pitch

> Tidekeepers là một daily strategy roguelite, nơi người chơi xây một đội hình để sống sót qua các đợt thủy triều được tạo từ dữ liệu thế giới thật.

### Một câu mô tả gameplay

> Mỗi ngày, đọc tín hiệu, xây đội hình, khóa lựa chọn và quay lại ngày mai để xem liệu đoàn Tidekeepers của bạn có sống sót hay không.

### Điểm khác biệt

- Dữ liệu thế giới thật tạo ra mỗi ngày khác nhau.
- Không cần chơi realtime.
- Không cần giao dịch tiền thật.
- Kết hợp daily game, roguelite và team-building.
- Thắng nhờ hiệu suất tương đối và quản trị rủi ro.
- Mỗi ngày là một trận chiến có thể giải thích.

---

## 45. Tagline

### Tagline chính

> **Build today. Face tomorrow.**

### Tagline thay thế

> **Read the tides. Keep the line.**

> **Every tide tells a story.**

> **Survive what tomorrow brings.**

---

## 46. Quyết định thiết kế khuyến nghị

Phiên bản đầu nên chốt theo hướng:

```text
Daily asynchronous
7-day Voyage
5 Keeper slots
Real-world basket data
Relative scoring
Fund Health
Capital riêng
No AI
No PvP
No realtime trading
No direct asset branding
```

Thứ cần kiểm chứng đầu tiên không phải dữ liệu thị trường có chính xác đến mức nào, mà là:

> Người chơi có cảm thấy việc đọc tín hiệu, chỉnh đội hình và quay lại ngày hôm sau đủ hấp dẫn để tiếp tục Voyage hay không?

Nếu vòng lặp này hoạt động, Tidekeepers có thể mở rộng sang:

- Voyage dài hơn.
- PvP bất đồng bộ.
- Guild.
- World events.
- Nhiều nguồn dữ liệu.
- Seasonal stories.
- Mobile app.
- Cosmetic economy.

---

## 47. Tóm tắt MVP

> **Tidekeepers** là game chiến thuật roguelite theo ngày. Người chơi xây một đội hình gồm các Keeper với vai trò, sector, kỹ năng và synergy khác nhau. Mỗi ngày, dữ liệu thế giới thật tạo ra một đợt Tide chung. Hệ thống chuẩn hóa biến động, so sánh đội hình với benchmark và áp dụng các luật gameplay để tính kết quả. Người chơi quay lại ngày hôm sau để xem breakdown, nhận phần thưởng, điều chỉnh chiến thuật và tiếp tục sống sót đến Final Tide.

