# Tidekeepers — Đặc tả tính toán 4 Sector

**Loại tài liệu:** Product, gameplay, data và backend engineering specification  
**Phạm vi:** Tính toán bốn Sector MVP: Crest, Ember, Current và Harbor  
**Mục tiêu triển khai:** Market ingestion, basket calculation, daily settlement và result breakdown  
**Backend mục tiêu:** Go + PostgreSQL  
**Trạng thái:** Draft sẵn sàng triển khai  

---

## 1. Mục đích

Tài liệu này định nghĩa đầy đủ cách **Tidekeepers** mô hình hóa, tính toán, lưu trữ, kiểm tra và sử dụng bốn Sector của MVP:

- **Crest**
- **Ember**
- **Current**
- **Harbor**

Sector có hai vai trò độc lập:

1. **Sector Identity** — phân loại Keeper, hiển thị fantasy, kích hoạt Synergy, Modifier và Objective.
2. **Sector Performance** — tạo benchmark theo ngày để so sánh công bằng các Keeper có đặc tính tương đồng.

Tài liệu bao gồm:

- Định nghĩa bốn Sector.
- Cách gán market component vào basket.
- Cách version basket mapping.
- Cách tính return của component và basket.
- Chuẩn hóa bằng Expected Turbulence.
- Cách tính Sector Benchmark.
- Cách tính Sector Relative Performance.
- Cách xếp hạng Keeper trong Sector.
- Cách chuyển kết quả Sector thành điểm gameplay.
- Chính sách dữ liệu thiếu và outlier.
- Data model PostgreSQL.
- Go domain interfaces.
- Job pipeline.
- API read model.
- Replay, audit và test.

Tài liệu **không chốt tên coin, market asset hoặc trọng số production cụ thể**. Những giá trị đó phải được publish bằng content version sau khi kiểm tra provider và balance bằng dữ liệu lịch sử.

---

## 2. Các ràng buộc từ gameplay hiện tại

Hệ thống Sector phải tuân thủ các nguyên tắc sau:

1. Một Keeper đại diện cho một **basket**, một phong cách hoặc một nhóm dữ liệu; không map trực tiếp với một coin duy nhất trong MVP.
2. Raw return không được trực tiếp quyết định sức mạnh Keeper.
3. Basket có Turbulence cao phải được chuẩn hóa bằng mức Turbulence kỳ vọng.
4. Keeper phải được so sánh với cả World Tide và Sector tương ứng.
5. Kết quả phải giải thích được cho người chơi.
6. Dữ liệu thị trường chỉ tạo điều kiện chiến đấu; Keeper, Synergy, Relic, Strategy, Modifier và Objective mới quyết định kết quả cuối.
7. Tất cả phép tính phải deterministic, versioned, idempotent và replayable.
8. Provider lỗi không được thay bằng random hoặc số 0.
9. Tất cả người chơi trong cùng Daily Tide phải dùng cùng market input, basket mapping, Sector benchmark và rule version.
10. Backend là nguồn dữ liệu authoritative; frontend không tự tính kết quả settlement.

---

## 3. Mô hình 4 Sector của MVP

| Sector | Fantasy gameplay | Đặc tính dữ liệu mong muốn | Điểm mạnh gameplay | Điểm yếu gameplay |
|---|---|---|---|---|
| Crest | Dòng chảy lớn, lâu đời và tương đối bền vững | Turbulence thấp đến trung bình, coverage tốt, có tính đại diện rộng | Ổn định tương đối, tham gia Broad Growth, hỗ trợ Vanguard | Có thể thua Ember trong speculative surge |
| Ember | Dòng chảy đầu cơ, momentum và Turbulence cao | Biên độ tăng giảm lớn, Depth lớn hơn, nhạy với Shock | Performance ceiling cao, Trickster và momentum | Pressure, crash và Drawdown Penalty |
| Current | Hạ tầng, mạng lưới và dòng chảy hệ thống | Turbulence trung bình, nhạy với Sector Rotation | Support, diversification, rotation | Yếu khi dòng tiền rời nhóm infrastructure |
| Harbor | Phòng thủ, bảo toàn và Turbulence thấp | Raw return nhỏ, Depth thấp hơn, thường tương đối tốt trong Broad Decline | Warden, Hull protection, Depth Control | Thường kém cạnh tranh trong Broad Growth mạnh |

Đây là phân loại gameplay, không phải khuyến nghị tài chính.

---

## 4. Thuật ngữ

| Thuật ngữ hệ thống | Thuật ngữ hiển thị | Ý nghĩa |
|---|---|---|
| Sector | Sector | Nhóm gameplay chứa nhiều Keeper basket tương đồng |
| Market asset | Hidden component | Thành phần dữ liệu thực nằm bên trong basket |
| Basket | Current | Nhóm component có trọng số đại diện cho Keeper |
| Raw return | Flow Change | Mức thay đổi trong cửa sổ Daily Tide |
| Expected volatility | Expected Turbulence | Mức biến động chuẩn dùng để normalize |
| Normalized performance | Normalized Flow | Hiệu suất đã điều chỉnh theo Turbulence |
| Sector benchmark | Sector Tide | Hiệu suất đại diện của toàn Sector trong ngày |
| Sector relative performance | Sector Advantage | Keeper performance trừ Sector benchmark |
| Global benchmark | World Tide | Benchmark chung của toàn game |
| Drawdown | Depth | Mức giảm lớn nhất từ đỉnh trong ngày |
| Recovery | Resurface | Khả năng hồi phục từ đáy |

---

## 5. Tổng quan pipeline

Pipeline authoritative:

```text
Provider observations
→ Component Returns
→ Keeper Basket Return
→ Expected Turbulence Normalization
→ Sector Eligibility
→ Sector Benchmark
→ Sector Relative Performance
→ Sector Percentile / Ranking
→ Sector Score
→ Keeper Result
→ Fleet Settlement
```

Các công thức cốt lõi:

```text
Component Return
= (Close Price - Open Price) / Open Price
```

```text
Basket Return
= Σ(Component Return × Effective Component Weight)
```

```text
Normalized Performance
= clamp(
    Basket Return / Expected Turbulence,
    -Normalization Cap,
    +Normalization Cap
  )
```

```text
Sector Benchmark
= weighted_average(
    Normalized Performance của các basket hợp lệ trong Sector
  )
```

```text
Sector Relative Performance
= Keeper Normalized Performance - Sector Benchmark
```

```text
Sector Score
= score_curve(Sector Relative Performance, Sector Ranking)
```

Giá trị đề xuất cho MVP:

```text
Normalization Cap = 2.0
Normalized Performance ∈ [-2.0, +2.0]
```

Mọi constant phải nằm trong content hoặc game rule version, không hard-code trong handler.

---

## 6. Sector Identity và Sector Performance

Sector Identity được cố định trong `keeper_definition_version`.

Ví dụ:

```text
Keeper: Crest Guardian
Sector: CREST
Basket mapping: crest_guardian_basket_v1
```

Sector Identity ảnh hưởng đến:

- Keeper card và icon.
- Synergy.
- Daily Modifier.
- Daily Objective.
- Sector benchmark membership.
- Result explanation.

Không được thay Sector của Keeper trong một Daily Tide đang chạy. Mọi thay đổi phải tạo:

- Keeper definition version mới.
- Basket mapping version mới nếu mapping thay đổi.
- Content version mới.

---

## 7. Định nghĩa Crest

### 7.1. Gameplay identity

Crest đại diện cho các Current lớn, phổ biến và tương đối bền vững.

### 7.2. Statistical profile mong muốn

- Expected Turbulence thấp hơn Ember.
- Provider coverage cao.
- Thanh khoản dữ liệu tốt.
- Không bị một component chi phối quá mức.
- Có khả năng phản ánh xu hướng chung nhưng không trùng hoàn toàn với World Tide.

### 7.3. Gameplay usage

- Vanguard.
- Generalist Growth.
- Broad-market participation.
- Giảm Turbulence toàn Fleet.
- Bảo vệ trong Broad Decline nhẹ.

### 7.4. Basket constraints đề xuất

```text
Minimum component count: 3
Preferred component count: 3–5
Maximum component weight: 50%
Minimum covered weight: 80%
```

---

## 8. Định nghĩa Ember

### 8.1. Gameplay identity

Ember đại diện cho các Current có momentum và Turbulence cao.

### 8.2. Statistical profile mong muốn

- Expected Turbulence cao nhất trong bốn Sector.
- Thường có raw return lớn theo cả hai chiều.
- Intraday Depth cao hơn.
- Nhạy với Broad Growth, Broad Decline và Shock.

### 8.3. Gameplay usage

- Trickster.
- Aggressive Growth.
- High ceiling.
- Risk/reward effect.
- Momentum multiplier.

### 8.4. Basket constraints đề xuất

```text
Minimum component count: 3
Preferred component count: 3–5
Maximum component weight: 40%
Minimum covered weight: 85%
```

Coverage của Ember nên cao vì việc thiếu một component Turbulence lớn có thể làm sai lệch basket đáng kể.

---

## 9. Định nghĩa Current

### 9.1. Gameplay identity

Current đại diện cho infrastructure, network và các hệ thống giúp giá trị hoặc dữ liệu lưu chuyển.

### 9.2. Statistical profile mong muốn

- Expected Turbulence trung bình.
- Nhạy với Sector Rotation.
- Có thể đi cùng World Tide hoặc tách khỏi World Tide theo từng giai đoạn.
- Không nên chỉ chứa một sub-theme quá hẹp.

### 9.3. Gameplay usage

- Support.
- Rotation.
- Diversification.
- Flow Transfer.
- Sector-relative objective.

### 9.4. Basket constraints đề xuất

```text
Minimum component count: 3
Preferred component count: 3–5
Maximum component weight: 45%
Minimum covered weight: 80%
```

---

## 10. Định nghĩa Harbor

### 10.1. Gameplay identity

Harbor đại diện cho các Current phòng thủ, bảo toàn và Turbulence thấp.

### 10.2. Statistical profile mong muốn

- Expected Turbulence thấp.
- Raw return thường nhỏ.
- Depth thấp hơn trong điều kiện bình thường.
- Có khả năng outperform tương đối trong Broad Decline.
- Thường kém World Tide trong Broad Growth mạnh.

### 10.3. Gameplay usage

- Warden.
- Defensive Vanguard.
- Hull protection.
- Drawdown reduction.
- Stability build.

### 10.4. Basket constraints đề xuất

```text
Minimum component count: 2
Preferred component count: 3–5
Maximum component weight: 45%
Maximum exceptional weight: 60%
Minimum covered weight: 90%
```

Harbor bắt buộc có `Turbulence Floor` để tránh chia cho một Expected Turbulence quá gần 0.

---

## 11. Keeper-to-Basket Mapping

Mỗi Keeper definition version phải tham chiếu đúng một basket mapping version.

```text
Keeper Definition Version
- id
- keeper_key
- content_version
- sector_key
- role_key
- base_risk
- basket_mapping_version_id
- passive_rule_key
- upgrade_tree
```

Basket mapping version:

```text
Basket Mapping Version
- id
- basket_key
- version
- sector_key
- calculation_method
- minimum_covered_weight
- normalization_cap
- expected_turbulence_policy_id
- benchmark_eligible
- status
- active_from
- active_to
```

Basket component:

```text
Basket Mapping Component
- basket_mapping_version_id
- market_asset_id
- target_weight
- minimum_observation_quality
- enabled
- sequence
```

Invariant:

```text
Σ target_weight của enabled components = 1.0
```

Đề xuất lưu weight dưới dạng integer:

```text
1.0 = 1,000,000 weight units
```

Không dùng `float64` làm authoritative value.

---

## 12. Quy tắc một basket cho nhiều Keeper

Nhiều Keeper có thể dùng chung một basket mapping nếu khác Role, Passive hoặc Upgrade.

Ví dụ:

```text
Crest Guardian → crest_core_v1
Crest Navigator → crest_core_v1
```

Trong trường hợp này:

- Basket metric chỉ được tính một lần.
- Sector benchmark chỉ đưa basket `crest_core_v1` vào một lần.
- Mỗi Keeper vẫn dùng cùng metric nhưng áp dụng rule riêng.

Không được tính lặp basket chỉ vì nhiều Keeper tham chiếu tới nó.

---

## 13. Cửa sổ dữ liệu Daily Tide

Mỗi Daily Tide định nghĩa một market window duy nhất:

```text
data_window_start
data_window_end
required_granularity
observation_tolerance
provider_id
```

Tất cả bốn Sector dùng cùng window.

Với mỗi component, hệ thống phải resolve:

- Open observation gần `data_window_start`.
- Close observation gần `data_window_end`.
- Intraday observations nếu cần tính Depth và Resurface.
- Provider timestamp.
- Raw response hash hoặc archive reference.
- Quality flags.

Ví dụ:

```text
Window start: 2026-07-30T00:00:00Z
Tolerance: ±5 phút
```

Observation ngoài tolerance bị xem là missing.

---

## 14. Tính Component Return

Công thức:

```text
component_return
= (close_price - open_price) / open_price
```

Validation bắt buộc:

- `open_price > 0`.
- `close_price > 0`.
- Cùng provider.
- Cùng market asset identifier.
- Timestamp nằm trong tolerance.
- Không chuyển missing thành 0.

Đề xuất precision:

```text
1 return unit = 100,000,000 return micros
```

Ví dụ:

```text
Open  = 100
Close = 104
Return = 0.04
Stored value = 4,000,000 micros
```

Rounding mode đề xuất:

```text
ROUND_HALF_AWAY_FROM_ZERO
```

Rounding mode phải thuộc game rule version.

---

## 15. Component Quality Status

Mỗi component calculation có một status:

```text
VALID
MISSING_OPEN
MISSING_CLOSE
OUTSIDE_TOLERANCE
INVALID_PRICE
OUTLIER_FLAGGED
PROVIDER_REJECTED
DISABLED_BY_MAPPING
```

Chỉ component `VALID` hoặc `OUTLIER_FLAGGED` được dùng, tùy outlier policy đã publish.

Raw provider evidence không được sửa hoặc xóa khi observation bị reject.

---

## 16. Coverage và Effective Weight

### 16.1. Khi đầy đủ dữ liệu

```text
effective_weight = target_weight
```

### 16.2. Khi thiếu component nhưng coverage vẫn đạt

```text
valid_covered_weight
= Σ(target_weight của VALID components)
```

Nếu:

```text
valid_covered_weight >= minimum_covered_weight
```

thì renormalize:

```text
effective_weight_i
= target_weight_i / valid_covered_weight
```

Ví dụ:

```text
A = 0.50
B = 0.30
C = 0.20

C missing
Coverage = 0.80
Threshold = 0.80
```

```text
A effective = 0.50 / 0.80 = 0.625
B effective = 0.30 / 0.80 = 0.375
```

Phải lưu:

- Target weight.
- Effective weight.
- Covered weight trước renormalization.
- Missing component IDs.
- Coverage decision.

### 16.3. Khi coverage không đạt

Nếu:

```text
valid_covered_weight < minimum_covered_weight
```

thì:

```text
basket status = DATA_INCOMPLETE
```

MVP không tự đặt basket thành neutral. Settlement tiếp tục ở `DATA_PENDING` cho tới khi:

- Provider retry thành công.
- Dữ liệu được repair.
- Một fallback provider policy đã được publish từ trước được áp dụng.

---

## 17. Tính Basket Return

```text
basket_return
= Σ(component_return_i × effective_weight_i)
```

Ví dụ:

| Component | Effective weight | Return | Contribution |
|---|---:|---:|---:|
| A | 0.50 | +4.00% | +2.00% |
| B | 0.30 | +2.00% | +0.60% |
| C | 0.20 | -1.00% | -0.20% |

```text
Basket Return = +2.40%
```

Phải persist từng component contribution, không chỉ final basket result.

---

## 18. Intraday Basket Series

Nếu provider có dữ liệu intraday, tạo index series cho basket.

Với mỗi component:

```text
component_index_t
= price_t / open_price
```

Basket index:

```text
basket_index_t
= Σ(component_index_t × effective_weight_i)
```

Tất cả component series phải được align vào cùng timestamp hoặc cùng bucket thời gian theo policy versioned.

Không interpolate tùy ý nếu policy chưa định nghĩa.

---

## 19. Tính Depth

```text
running_peak_t
= max(basket_index_0 ... basket_index_t)
```

```text
depth_t
= (basket_index_t - running_peak_t) / running_peak_t
```

```text
max_depth
= minimum(depth_t)
```

`max_depth` luôn bằng 0 hoặc âm.

Ví dụ:

```text
Peak = 1.10
Low  = 0.99
Depth = (0.99 - 1.10) / 1.10 = -10%
```

---

## 20. Tính Resurface

```text
window_low
= minimum(basket_index_t)
```

```text
resurface_from_low
= (basket_index_close - window_low) / window_low
```

Resurface dùng cho:

- Recovery score.
- Contrarian Keeper.
- Recovery Tide Modifier.
- Market Regime classification.

---

## 21. Realized Turbulence

Exact method phải versioned.

Phương án MVP đề xuất:

```text
interval_log_return_t
= ln(index_t / index_t-1)
```

```text
realized_turbulence
= sqrt(Σ(interval_log_return_t²))
```

Không annualize vì metric chỉ dùng trong một Daily Tide.

Đây là quyết định kỹ thuật đề xuất; gameplay design hiện tại chưa bắt buộc một công thức realized volatility cụ thể.

---

## 22. Expected Turbulence

Expected Turbulence là baseline dùng để normalize basket.

```text
normalized_performance
= basket_return / expected_turbulence
```

Expected Turbulence phải:

- Lớn hơn 0.
- Được version.
- Được chốt trước khi Daily Tide mở.
- Không thay đổi trong lúc Tide đang chạy.
- Có source hoặc policy rõ ràng.

### 22.1. Policy types hỗ trợ

```text
STATIC_CONTENT_VALUE
ROLLING_HISTORICAL_MEDIAN
ROLLING_EXPONENTIAL_AVERAGE
```

### 22.2. Đề xuất MVP

Dùng:

```text
STATIC_CONTENT_VALUE
```

Giá trị được tính offline từ dữ liệu lịch sử, kiểm tra, sau đó publish thành content version mới.

Lợi ích:

- Dễ audit.
- Không thay đổi kỳ vọng giữa Tide.
- Dễ replay.
- Dễ balance.

### 22.3. Turbulence Floor

```text
effective_expected_turbulence
= max(expected_turbulence, turbulence_floor)
```

Harbor đặc biệt cần floor để tránh một raw return nhỏ tạo normalized value cực lớn.

---

## 23. Normalize Basket Performance

```text
normalized_performance
= clamp(
    basket_return / effective_expected_turbulence,
    -normalization_cap,
    +normalization_cap
  )
```

MVP default đề xuất:

```text
normalization_cap = 2.0
```

Ví dụ:

| Basket | Raw Return | Expected Turbulence | Normalized |
|---|---:|---:|---:|
| Crest | +4% | 4% | +1.00 |
| Ember | +12% | 15% | +0.80 |
| Harbor | +0.2% | 0.5% | +0.40 |

Ember có raw return lớn nhất nhưng không tự động có normalized performance cao nhất.

---

## 24. Điều kiện tham gia Sector Benchmark

Một basket chỉ được tham gia benchmark khi:

1. Mapping version được publish cho Daily Tide.
2. Sector của mapping trùng Sector benchmark.
3. Basket metric status là `READY`.
4. Coverage đạt threshold.
5. Expected Turbulence hợp lệ.
6. Normalized Performance tính thành công.
7. `benchmark_eligible = true`.

Sector benchmark được tính từ toàn bộ basket hợp lệ của game, **không phải từ Keeper mà người chơi chọn**.

Điều này đảm bảo:

- Popular Keeper không làm benchmark lệch.
- Tất cả người chơi dùng cùng Sector Tide.
- Benchmark không phụ thuộc meta đội hình.

---

## 25. Tính Sector Benchmark

### 25.1. Equal Weight — đề xuất MVP

```text
sector_benchmark
= average(normalized_performance của eligible unique baskets)
```

Ví dụ Crest:

```text
Crest basket A = +1.00
Crest basket B = +0.60
Crest basket C = +0.20
```

```text
Crest Benchmark
= (+1.00 + 0.60 + 0.20) / 3
= +0.60
```

### 25.2. Configured Weight — hỗ trợ tương lai

```text
sector_benchmark
= Σ(normalized_performance_i × benchmark_weight_i)
```

```text
Σ benchmark_weight_i = 1.0
```

MVP nên dùng Equal Weight vì:

- Dễ giải thích.
- Dễ balance.
- Tránh lặp weighting hai lần: một lần trong basket và một lần trong Sector.

### 25.3. Minimum Sector Coverage

Đề xuất:

```text
minimum eligible unique baskets per Sector = 2
```

Nếu không đạt:

```text
sector benchmark status = DATA_INCOMPLETE
```

Settlement không được tự fallback về World Tide nếu rule version chưa định nghĩa.

---

## 26. Tính Sector Relative Performance

```text
sector_relative_performance
= keeper_normalized_performance - sector_benchmark
```

Ví dụ:

```text
Keeper Normalized Performance = +1.00
Crest Benchmark               = +0.60
Sector Relative Performance   = +0.40
```

Ý nghĩa:

- `> 0`: Keeper tốt hơn nhóm tương đồng.
- `= 0`: Keeper bằng mức trung bình Sector.
- `< 0`: Keeper kém hơn Sector.

Một Keeper có raw return dương vẫn có thể nhận Sector Relative âm.

---

## 27. Sector Ranking và Percentile

Các eligible basket được sort theo `normalized_performance` tăng dần.

### 27.1. Percentile

Với `n > 1`:

```text
percentile
= rank_index / (n - 1)
```

Trong đó `rank_index` bắt đầu từ 0.

Với `n = 1`:

```text
percentile = 0.5
```

Tuy nhiên MVP không nên cho Sector chỉ có một basket hợp lệ.

### 27.2. Tie handling

Các basket bằng nhau dùng average rank.

Ví dụ:

```text
Values: -0.2, +0.3, +0.3, +0.8
```

Hai basket `+0.3` cùng dùng rank trung bình của vị trí 1 và 2.

### 27.3. Centered Rank

```text
centered_sector_rank
= percentile × 2 - 1
```

Kết quả:

```text
lowest  = -1.0
middle  ≈  0.0
highest = +1.0
```

---

## 28. Chuyển Sector Performance thành Sector Score

Gameplay design quy định `Sector Ranking` chiếm 20% Daily Score nhưng chưa chốt score curve. Tài liệu này đề xuất công thức MVP deterministic.

### 28.1. Relative Component

```text
relative_component
= clamp(
    sector_relative_performance / sector_relative_scale,
    -1,
    +1
  )
```

Đề xuất:

```text
sector_relative_scale = 0.50 normalized units
```

### 28.2. Rank Component

```text
rank_component
= centered_sector_rank
```

### 28.3. Blend

```text
sector_score_normalized
= 70% × relative_component
+ 30% × rank_component
```

```text
sector_score_normalized ∈ [-1, +1]
```

Lý do ưu tiên 70% relative:

- Phản ánh khoảng cách thực tế với Sector benchmark.
- Không biến Sector thành winner-take-all.
- Giảm độ thô khi mỗi Sector chỉ có 2–3 basket.

### 28.4. Score Points

Nếu component scale là `[-100, +100]`:

```text
sector_score_points
= round(sector_score_normalized × 100)
```

### 28.5. Daily Contribution

```text
sector_daily_contribution
= sector_score_points × 20%
```

Theo scaled weight:

```text
sector_ranking_weight = 2000
weight_denominator    = 10000
```

```text
sector_contribution_microscore
= sector_score_points_microscore
  × sector_ranking_weight
  / weight_denominator
```

---

## 29. Ví dụ đầy đủ cho Ember

Giả sử Ember có ba basket:

| Basket | Raw Return | Expected Turbulence | Normalized |
|---|---:|---:|---:|
| Ember Trickster | +12% | 15% | +0.80 |
| Flame Runner | +6% | 12% | +0.50 |
| Ash Gambler | -3% | 10% | -0.30 |

### 29.1. Ember Benchmark

```text
Ember Benchmark
= (+0.80 + 0.50 - 0.30) / 3
= +0.333333
```

### 29.2. Sector Relative

| Basket | Normalized | Benchmark | Sector Relative |
|---|---:|---:|---:|
| Ember Trickster | +0.80 | +0.3333 | +0.4667 |
| Flame Runner | +0.50 | +0.3333 | +0.1667 |
| Ash Gambler | -0.30 | +0.3333 | -0.6333 |

### 29.3. Relative Component với scale 0.50

```text
Ember Trickster
= clamp(0.4667 / 0.50)
= +0.9334
```

```text
Flame Runner
= clamp(0.1667 / 0.50)
= +0.3334
```

```text
Ash Gambler
= clamp(-0.6333 / 0.50)
= -1.0000
```

### 29.4. Centered Rank

```text
Ash Gambler     = -1.0
Flame Runner    =  0.0
Ember Trickster = +1.0
```

### 29.5. Sector Score

```text
Ember Trickster
= 0.70 × 0.9334 + 0.30 × 1.0
= 0.9534
≈ +95 points
```

```text
Flame Runner
= 0.70 × 0.3334 + 0.30 × 0.0
= 0.2334
≈ +23 points
```

```text
Ash Gambler
= 0.70 × -1.0 + 0.30 × -1.0
= -1.0
= -100 points
```

Daily Score contribution trước các rule khác:

```text
Ember Trickster Sector Contribution
= 95 × 20%
= 19 points
```

---

## 30. Fleet-Level Sector Score

Gameplay hiện tại chưa chốt cách aggregate Sector Score của nhiều Keeper. Đề xuất MVP:

```text
fleet_sector_score
= average(sector_score của các occupied slots)
```

Rules:

- Empty slot không tham gia mẫu số.
- Keeper instance không được xuất hiện hai slot.
- Hai instance cùng Keeper definition được tính riêng nếu duplicate hợp lệ.
- Invalid snapshot làm settlement fail cho player đó; không silently bỏ Keeper.
- Synergy được áp dụng ở rule stage sau, không sửa historical Sector benchmark.

Ví dụ:

```text
Slot 1: +80
Slot 2: +20
Slot 3: -40
```

```text
Fleet Sector Score
= (80 + 20 - 40) / 3
= +20
```

```text
Sector Contribution
= +20 × 20%
= +4 Daily Score points
```

Nếu sau này dùng slot weight hoặc diminishing return, phải tạo game rule version mới.

---

## 31. Quan hệ với Keeper Market Score

Gameplay design có công thức:

```text
Keeper Market Score
= 40% Global Relative Score
+ 40% Sector Relative Score
+ 20% Risk-adjusted Score
```

Đồng thời Daily Score có công thức:

```text
Daily Score
= 40% Relative Performance
+ 20% Sector Ranking
+ 15% Depth Control
+ 10% Recovery
+ 10% Synergy & Skill Effects
+ 5% Daily Objective
```

Hai công thức này chưa được tài liệu nguồn ghép thành một pipeline duy nhất. Đề xuất MVP:

- Ba lớp của Keeper dùng để tạo **Keeper market analysis**.
- Global layer feed vào `Relative Performance`.
- Sector layer feed vào `Sector Ranking`.
- Risk-adjusted layer hỗ trợ normalization, Pressure và các risk rule.
- Không cộng thêm Keeper Market Score một lần nữa vào Daily Score để tránh double counting.

Quyết định này cần được chốt trước khi code settlement engine.

---

## 32. Daily Modifier và Sector

Daily Modifier không sửa raw market metric đã lưu. Modifier chỉ áp dụng effect ở settlement rule stage.

### Rotation Day

Có thể thay đổi weight:

```text
sector_ranking_weight tăng
relative_performance_weight giảm tương ứng
```

Tổng weight vẫn phải bằng 100%.

### Flight to Harbor

```text
Warden và Vanguard nhận defensive effect.
Ember hoặc Trickster nhận thêm Pressure.
```

Không sửa Harbor Benchmark hoặc Ember Benchmark.

### Wild Current

```text
Keeper có |normalized performance| lớn nhận bonus.
Depth Penalty tăng.
```

Rule phải emit contribution riêng.

---

## 33. Synergy và Sector

Sector Synergy được tính từ locked Fleet snapshot.

Ví dụ:

```text
2 Crest:
Giảm Turbulence toàn đội.
```

```text
2 Ember:
Tăng Performance Ceiling và Risk.
```

```text
2 Harbor:
Giảm Hull damage.
```

```text
Diversified Fleet:
Có 4 Sector khác nhau thì giảm Sector Concentration Penalty.
```

Synergy có thể sửa:

- Score contribution.
- Depth penalty.
- Pressure.
- Hull damage.
- Score cap.
- Reward condition.

Synergy không được sửa:

- Provider observations.
- Basket return đã persist.
- Sector benchmark.
- World Tide benchmark.

---

## 34. Sector Concentration

Đề xuất thêm metric Fleet-level:

```text
sector_count_s
= số Keeper trong Fleet thuộc Sector s
```

```text
sector_concentration_ratio
= max(sector_count_s) / occupied_slot_count
```

Ví dụ Fleet 5 slot có 4 Ember:

```text
concentration_ratio = 4 / 5 = 0.80
```

Metric này dùng cho:

- Concentration Audit Modifier.
- Diversified Fleet Synergy.
- Risk penalty.

Metric không trực tiếp sửa Sector Benchmark.

---

## 35. Sector Inputs cho Market Regime

Bốn Sector Benchmark có thể cung cấp input cho Regime classifier.

```text
sector_positive_breadth
= count(sector_benchmark > positive_threshold) / 4
```

```text
sector_negative_breadth
= count(sector_benchmark < negative_threshold) / 4
```

```text
sector_dispersion
= max(sector_benchmarks) - min(sector_benchmarks)
```

```text
sector_rotation_strength
= standard_deviation(sector_benchmarks)
```

Ví dụ sử dụng:

- Positive breadth cao + World Tide dương → Broad Growth.
- Negative breadth cao + World Tide âm → Broad Decline.
- Mixed signs + dispersion cao → Rotation.
- Absolute values thấp + dispersion thấp → Sideways.

Threshold cụ thể thuộc `market_regime_rule_version`.

---

## 36. Database Schema

### 36.1. `sectors`

```text
id
sector_key             unique
name_key
summary_key
status
created_at
updated_at
```

MVP rows:

```text
CREST
EMBER
CURRENT
HARBOR
```

### 36.2. `sector_definition_versions`

```text
id
sector_id
content_version
benchmark_method
minimum_eligible_baskets
relative_scale_units
relative_blend_weight
rank_blend_weight
score_cap_units
active_from
active_to
status
created_at
```

Unique:

```text
unique(sector_id, content_version)
```

### 36.3. `basket_mapping_versions`

```text
id
basket_key
version
sector_definition_version_id
calculation_method
minimum_covered_weight_units
expected_turbulence_policy_id
normalization_cap_units
benchmark_eligible
status
active_from
active_to
created_at
```

Unique:

```text
unique(basket_key, version)
```

### 36.4. `basket_mapping_components`

```text
id
basket_mapping_version_id
market_asset_id
target_weight_units
minimum_quality
sequence
enabled
created_at
```

Unique:

```text
unique(basket_mapping_version_id, market_asset_id)
unique(basket_mapping_version_id, sequence)
```

### 36.5. `expected_turbulence_policies`

```text
id
policy_key
version
policy_type
lookback_days nullable
static_value_units nullable
floor_units
rounding_mode
status
created_at
```

### 36.6. `basket_metrics`

```text
id
daily_tide_id
basket_mapping_version_id
status
covered_weight_units
raw_return_units
expected_turbulence_units
normalized_performance_units
max_depth_units nullable
resurface_units nullable
realized_turbulence_units nullable
input_checksum
calculated_at
created_at
```

Unique:

```text
unique(daily_tide_id, basket_mapping_version_id)
```

### 36.7. `basket_metric_components`

```text
id
basket_metric_id
market_asset_id
status
target_weight_units
effective_weight_units
open_observation_id nullable
close_observation_id nullable
component_return_units nullable
contribution_units nullable
quality_flags
created_at
```

### 36.8. `sector_benchmarks`

```text
id
daily_tide_id
sector_definition_version_id
status
calculation_method
eligible_basket_count
benchmark_units
minimum_required_baskets
input_checksum
calculated_at
created_at
```

Unique:

```text
unique(daily_tide_id, sector_definition_version_id)
```

### 36.9. `sector_benchmark_members`

```text
id
sector_benchmark_id
basket_metric_id
benchmark_weight_units
normalized_performance_units
contribution_units
rank_index
rank_percentile_units
created_at
```

Unique:

```text
unique(sector_benchmark_id, basket_metric_id)
```

### 36.10. `keeper_sector_results`

```text
id
daily_result_id
keeper_instance_id
basket_metric_id
sector_benchmark_id
sector_relative_units
sector_percentile_units
relative_component_units
rank_component_units
sector_score_units
sector_contribution_units
created_at
```

---

## 37. Content Configuration mẫu

Chỉ dùng fictional keys; không phải production asset list.

```yaml
contentVersion: sector-mvp-v1

sectors:
  - key: CREST
    benchmarkMethod: EQUAL_WEIGHT
    minimumEligibleBaskets: 2
    relativeScale: 0.50
    relativeBlendWeight: 0.70
    rankBlendWeight: 0.30

  - key: EMBER
    benchmarkMethod: EQUAL_WEIGHT
    minimumEligibleBaskets: 2
    relativeScale: 0.50
    relativeBlendWeight: 0.70
    rankBlendWeight: 0.30

  - key: CURRENT
    benchmarkMethod: EQUAL_WEIGHT
    minimumEligibleBaskets: 2
    relativeScale: 0.50
    relativeBlendWeight: 0.70
    rankBlendWeight: 0.30

  - key: HARBOR
    benchmarkMethod: EQUAL_WEIGHT
    minimumEligibleBaskets: 2
    relativeScale: 0.50
    relativeBlendWeight: 0.70
    rankBlendWeight: 0.30

baskets:
  - key: crest_guardian
    sector: CREST
    minimumCoveredWeight: 0.80
    normalizationCap: 2.00
    benchmarkEligible: true
    expectedTurbulence:
      type: STATIC_CONTENT_VALUE
      value: 0.0400
      floor: 0.0025
    components:
      - assetKey: CREST_A
        weight: 0.40
      - assetKey: CREST_B
        weight: 0.35
      - assetKey: CREST_C
        weight: 0.25

  - key: ember_trickster
    sector: EMBER
    minimumCoveredWeight: 0.85
    normalizationCap: 2.00
    benchmarkEligible: true
    expectedTurbulence:
      type: STATIC_CONTENT_VALUE
      value: 0.1500
      floor: 0.0100
    components:
      - assetKey: EMBER_A
        weight: 0.35
      - assetKey: EMBER_B
        weight: 0.35
      - assetKey: EMBER_C
        weight: 0.30
```

---

## 38. Go Domain Types

```go
type SectorKey string

const (
    SectorCrest   SectorKey = "CREST"
    SectorEmber   SectorKey = "EMBER"
    SectorCurrent SectorKey = "CURRENT"
    SectorHarbor  SectorKey = "HARBOR"
)
```

### 38.1. BasketCalculator

```go
type BasketCalculator interface {
    Calculate(
        ctx context.Context,
        input BasketCalculationInput,
    ) (BasketMetric, error)
}
```

```go
type BasketCalculationInput struct {
    DailyTideID           string
    Mapping               BasketMappingVersion
    ComponentObservations []ComponentObservationWindow
    Precision             PrecisionPolicy
}
```

```go
type BasketMetric struct {
    BasketMappingVersionID string
    Sector                  SectorKey
    Status                  BasketMetricStatus
    CoveredWeight           int64
    RawReturn               int64
    ExpectedTurbulence      int64
    NormalizedPerformance   int64
    MaxDepth                *int64
    Resurface               *int64
    RealizedTurbulence      *int64
    Components              []BasketComponentMetric
    InputChecksum           string
}
```

### 38.2. SectorBenchmarkCalculator

```go
type SectorBenchmarkCalculator interface {
    Calculate(
        ctx context.Context,
        input SectorBenchmarkInput,
    ) (SectorBenchmark, error)
}
```

```go
type SectorBenchmarkInput struct {
    DailyTideID string
    Definition  SectorDefinitionVersion
    Baskets     []BasketMetric
    Precision   PrecisionPolicy
}
```

```go
type SectorBenchmark struct {
    Sector              SectorKey
    Status              SectorBenchmarkStatus
    EligibleBasketCount int
    Benchmark           int64
    Members             []SectorBenchmarkMember
    InputChecksum       string
}
```

### 38.3. SectorScoreCalculator

```go
type SectorScoreCalculator interface {
    Score(input SectorScoreInput) (SectorScoreOutput, error)
}
```

```go
type SectorScoreInput struct {
    KeeperNormalizedPerformance int64
    SectorBenchmark             int64
    SectorPercentile            int64
    RelativeScale               int64
    RelativeBlendWeight         int64
    RankBlendWeight             int64
    ScoreCap                    int64
    Precision                   PrecisionPolicy
}
```

```go
type SectorScoreOutput struct {
    RelativePerformance int64
    RelativeComponent   int64
    RankComponent       int64
    NormalizedScore     int64
    ScorePoints         int64
}
```

---

## 39. Pure Function Requirements

Các function tính toán Sector phải:

- Không truy cập database.
- Không gọi network.
- Không đọc current time.
- Không dùng random.
- Chỉ dùng immutable input.
- Dùng scaled integer hoặc exact decimal.
- Trả typed error ổn định.
- Cho cùng kết quả dù input slice có thứ tự khác, sau canonical sorting.

Repository structure đề xuất:

```text
internal/market/basket
internal/market/sector
internal/market/precision
internal/settlement/engine
```

---

## 40. Basket Calculation Job

Job:

```text
calculate_basket_metrics
```

Với mỗi basket mapping version:

1. Acquire idempotent scope lock.
2. Load component observations.
3. Validate observation quality.
4. Tính valid covered weight.
5. Renormalize effective weights nếu cần.
6. Tính component returns.
7. Tính basket return.
8. Resolve Expected Turbulence.
9. Normalize và clamp.
10. Tính Depth, Resurface và Turbulence nếu required.
11. Persist aggregate và component evidence.
12. Mark status.

Idempotency scope:

```text
calculate_basket_metrics:{daily_tide_id}:{basket_mapping_version_id}
```

---

## 41. Sector Benchmark Job

Job:

```text
calculate_sector_benchmarks
```

Với mỗi Sector:

1. Acquire PostgreSQL advisory lock theo Daily Tide và Sector.
2. Load tất cả benchmark-eligible basket metrics.
3. Deduplicate theo basket mapping version.
4. Validate minimum eligible basket count.
5. Canonical sort theo basket mapping version ID.
6. Tính benchmark.
7. Tính rank và percentile.
8. Persist benchmark members.
9. Tạo input checksum.
10. Mark Sector benchmark `READY`.

Idempotency scope:

```text
calculate_sector_benchmark:{daily_tide_id}:{sector_key}:{rule_version}
```

Settlement chỉ được chạy khi:

```text
All required Basket Metrics = READY
CREST Benchmark             = READY
EMBER Benchmark             = READY
CURRENT Benchmark           = READY
HARBOR Benchmark            = READY
World Tide Benchmark        = READY
Market Regime               = READY
```

---

## 42. Settlement Integration

Settlement input cho mỗi Keeper phải chứa:

```text
keeper_instance
keeper_definition_version
basket_metric
sector_benchmark
sector_rank
sector_percentile
strategy
relics
synergies
daily_modifier
daily_objective
game_rule_version
```

Rule order liên quan Sector:

```text
1. Load immutable Basket Metric
2. Load immutable Sector Benchmark
3. Calculate Sector Relative Performance
4. Calculate Sector Score
5. Aggregate Fleet Sector Score
6. Apply Keeper Upgrade modifiers
7. Apply Keeper Passive rules
8. Apply Synergy rules
9. Apply Relic rules
10. Apply Strategy
11. Apply Daily Modifier
12. Apply Objective
13. Apply clamps and caps
14. Calculate Hull and Supplies effects
```

Không được cho passive rule sửa lại provider input hoặc Sector benchmark.

---

## 43. Contribution Records

Mỗi Keeper cần contribution rows đủ để giải thích kết quả:

```text
source_type
source_key
source_instance_id
component_key
raw_value
benchmark_value
relative_value
rank_percentile
score_amount
explanation_key
explanation_params
sequence
```

Ví dụ:

```text
source_type: KEEPER_MARKET
source_key: CREST_GUARDIAN
component_key: SECTOR_RANKING
raw_value: 1.0000
benchmark_value: 0.6000
relative_value: 0.4000
rank_percentile: 1.0000
score_amount: +16.0000
explanation_key: result.sector.outperformed
params:
  sector: CREST
  rank: 1
  total: 3
```

---

## 44. Checksum và Replay

### 44.1. Basket Metric Checksum

Canonical input:

- Daily Tide ID.
- Basket mapping version ID.
- Component IDs.
- Open/close observation IDs.
- Target/effective weights.
- Component returns.
- Expected Turbulence policy version.
- Precision policy.
- Normalization cap.

### 44.2. Sector Benchmark Checksum

Canonical input:

- Daily Tide ID.
- Sector definition version ID.
- Benchmark method.
- Ordered basket metric IDs.
- Normalized performance values.
- Benchmark weights.
- Minimum eligible rule.
- Precision và rounding policy.

Replay phải tạo đúng:

- Numerical output.
- Member order.
- Percentiles.
- Checksum.

---

## 45. API Read Model

Daily context chỉ cần trả summary phục vụ UI.

```json
{
  "sectorTides": [
    {
      "sector": "CREST",
      "displayName": "Crest",
      "strengthBand": "STRONG",
      "benchmarkScore": "0.6200",
      "direction": "POSITIVE"
    },
    {
      "sector": "EMBER",
      "displayName": "Ember",
      "strengthBand": "VOLATILE",
      "benchmarkScore": "-0.1800",
      "direction": "MIXED"
    }
  ]
}
```

Result breakdown:

```json
{
  "keeperId": "ki_123",
  "sector": "CREST",
  "normalizedPerformance": "1.0000",
  "sectorBenchmark": "0.6000",
  "sectorRelativePerformance": "0.4000",
  "sectorPercentile": "1.0000",
  "sectorRank": 1,
  "sectorMemberCount": 3,
  "sectorScore": 80,
  "sectorContribution": "16.0000"
}
```

Nên dùng decimal string hoặc scaled integer trong API với authoritative value.

Main gameplay UI không cần hiển thị tên asset thực. Nếu có transparency page, endpoint riêng có thể mô tả methodology.

---

## 46. Frontend Presentation

### 46.1. Preparation

Hiển thị:

- Sector identity của Keeper.
- Sector Synergy.
- Signal theo Sector.
- Daily Modifier ảnh hưởng Sector nào.

Không hiển thị final Sector Score khi Tide chưa đóng.

### 46.2. Result

Ví dụ:

```text
Crest Guardian
Normalized Flow       +1.00
Crest Sector Tide     +0.60
Sector Advantage      +0.40
Sector Rank           1 / 3
Sector Contribution   +16
```

Explanation:

```text
Crest Guardian hoạt động tốt hơn các Current khác trong Crest Sector trong Tide này.
```

Không dùng câu mô tả như dự báo hoặc khuyến nghị đầu tư.

---

## 47. Error Taxonomy

Internal errors:

```text
SECTOR_UNKNOWN
SECTOR_DEFINITION_INVALID
BASKET_MAPPING_INVALID
BASKET_WEIGHT_SUM_INVALID
BASKET_DATA_INCOMPLETE
COMPONENT_PRICE_INVALID
EXPECTED_TURBULENCE_INVALID
EXPECTED_TURBULENCE_TOO_LOW
NORMALIZATION_CONFIG_INVALID
SECTOR_MINIMUM_BASKETS_NOT_MET
SECTOR_BENCHMARK_INPUT_INVALID
SECTOR_SCORE_CONFIG_INVALID
PRECISION_OVERFLOW
CHECKSUM_MISMATCH
```

Player-facing errors:

```text
PROVIDER_DATA_PENDING
RESULT_NOT_READY
```

Không expose provider details hoặc hidden component identifiers cho player.

---

## 48. Outlier Policy

Raw data luôn được lưu.

Policy types hỗ trợ:

```text
REJECT_COMPONENT
ACCEPT_AND_CLAMP_GAMEPLAY_INFLUENCE
REQUIRE_OPERATOR_REVIEW
```

MVP đề xuất:

1. Reject invalid schema và non-positive price.
2. Flag return bất thường theo asset/provider threshold.
3. Nếu outlier ảnh hưởng coverage đáng kể, giữ basket pending.
4. Operator kiểm tra và approve data hoặc repair source.
5. Clamp normalized gameplay influence, không sửa raw source value.

Không được:

- Thay outlier bằng zero.
- Thay bằng previous close mà không có policy.
- Xóa raw observation.

---

## 49. Precision Policy

Đề xuất:

```text
Price                    PostgreSQL NUMERIC hoặc provider decimal
Component return         1e-8
Weight                   1e-6
Normalized performance   1e-6
Percentile               1e-6
Score                    1e-4
```

Go implementation có thể dùng:

- `math/big.Int` hoặc `math/big.Rat`.
- Fixed-point decimal library đã review.
- `int64` với checked arithmetic và `big.Int` cho intermediate multiplication.

Không dùng `float64` equality cho authoritative settlement.

---

## 50. Content Publication Validation

Trước khi publish content:

1. Có đúng bốn Sector MVP.
2. Mọi Keeper tham chiếu Sector hợp lệ.
3. Mọi Keeper tham chiếu basket mapping version hợp lệ.
4. Keeper Sector trùng Basket Sector, trừ exception đã khai báo.
5. Enabled component weights sum đúng 1.0.
6. Không có weight bằng 0 hoặc âm.
7. Maximum component weight không vượt Sector policy nếu không có approval.
8. Minimum covered weight nằm trong `(0, 1]`.
9. Expected Turbulence lớn hơn hoặc bằng floor.
10. Normalization cap lớn hơn 0.
11. Relative blend + Rank blend = 1.0.
12. Mỗi Sector có đủ minimum benchmark-eligible baskets.
13. Market asset references tồn tại ở provider.
14. Upgrade và Keeper references hợp lệ.
15. Replay fixtures pass.
16. Historical balance simulation pass.

Published version là immutable.

---

## 51. Balance Validation

Chạy simulation trên historical Tide windows và theo dõi:

- Mean normalized performance theo Sector.
- Standard deviation theo Sector.
- Tỷ lệ chạm `±2.0` cap.
- Phân phối Sector benchmark.
- Phân phối Sector Relative.
- Tần suất Sector đứng đầu và cuối.
- Correlation giữa Raw Return và Sector Score.
- Harbor performance trong Broad Decline.
- Ember upside/downside symmetry.
- Crest dominance risk.
- Current rotation sensitivity.

Balance alarms đề xuất:

```text
Một Sector đứng đầu > 40% số ngày sample.
Một Sector đứng cuối > 40% số ngày sample.
> 10% normalized values chạm cap.
Harbor gần như không bao giờ dương trong decline sample.
Ember có average score dương trong symmetric sample.
Một basket giải thích > 60% variance của Sector benchmark.
```

Threshold này chỉ là điểm bắt đầu để balance, không phải final rule.

---

## 52. Unit Tests

### Component và Basket

- Positive return.
- Negative return.
- Zero return.
- Invalid zero open price.
- Full coverage.
- Missing component nhưng coverage pass.
- Missing component và coverage fail.
- Effective weights sum đúng 1.0.
- Positive normalization cap.
- Negative normalization cap.
- Turbulence floor.
- Exact rounding boundary.

### Sector Benchmark

- Equal-weight average.
- Configured-weight average.
- Duplicate basket mapping bị reject.
- Minimum eligible baskets fail.
- Input ordering không đổi output.
- Tied ranking dùng average rank.
- Percentile cho 2, 3 và nhiều basket.

### Sector Score

- Keeper bằng benchmark.
- Keeper cao hơn benchmark.
- Keeper thấp hơn benchmark.
- Relative component clamp.
- Rank bounds.
- Blend weight validation.
- Exact score conversion.

---

## 53. Property và Invariant Tests

```text
Effective weights của READY basket luôn sum thành 1.0.
```

```text
Sector benchmark nằm giữa min và max normalized values
khi benchmark weights không âm.
```

```text
Với equal weights, trung bình Sector Relative bằng 0
trước rounding.
```

```text
Sector percentile luôn nằm trong [0, 1].
```

```text
Sector normalized score luôn nằm trong [-1, +1].
```

```text
Cùng input luôn tạo cùng output và checksum.
```

```text
Đổi thứ tự input không đổi benchmark.
```

```text
Một basket mapping chỉ xuất hiện một lần trong Sector benchmark.
```

---

## 54. Integration Tests

1. Publish content version chứa bốn Sector.
2. Reject basket có weight sum sai.
3. Ingest observations và tính basket metrics.
4. Retry basket job không tạo duplicate.
5. Tính đủ bốn Sector benchmark.
6. Retry Sector job không tạo duplicate.
7. Settlement bị block khi một Sector incomplete.
8. Repair data rồi settlement tiếp tục.
9. Replay Sector benchmark khớp checksum.
10. Keeper result reference đúng Sector benchmark version.
11. Concurrent workers không tạo hai benchmark rows.
12. Content version mới không thay đổi historical Tide.
13. Hai Keeper dùng cùng basket không làm benchmark duplicate.

---

## 55. Replay Fixtures

Tối thiểu:

```text
crest_outperforms_broad_growth
harbor_outperforms_broad_decline
ember_surges_but_normalizes_below_crest
ember_crash_hits_negative_cap
current_leads_rotation
all_sectors_sideways
one_component_missing_coverage_passes
coverage_failure_blocks_sector
sector_tie_ranking
outlier_requires_review
```

Mỗi fixture lưu:

- Content version.
- Market observations.
- Expected component returns.
- Expected basket returns.
- Expected normalized values.
- Expected Sector benchmarks.
- Expected ranks và percentiles.
- Expected Sector Scores.
- Exact checksums.

---

## 56. Observability

Metrics:

```text
basket_calculation_total{sector,status}
basket_coverage_ratio{basket}
basket_normalization_cap_total{sector,direction}
sector_benchmark_calculation_total{sector,status}
sector_eligible_basket_count{sector}
sector_benchmark_value{sector}
sector_calculation_duration_seconds
sector_replay_mismatch_total{sector}
```

Structured log fields:

```text
daily_tide_id
sector_key
basket_mapping_version_id
content_version
rule_version
job_id
status
covered_weight
input_checksum
error_code
```

Không log provider credentials hoặc raw response body chứa secret.

---

## 57. Admin CLI

```text
tidekeepers-admin inspect-sector \
  --daily-tide-id <id> \
  --sector CREST
```

```text
tidekeepers-admin replay-sector \
  --sector-benchmark-id <id>
```

```text
tidekeepers-admin validate-sector-content \
  --content-version sector-mvp-v1
```

```text
tidekeepers-admin export-sector-evidence \
  --daily-tide-id <id>
```

Mọi repair mutation yêu cầu:

- Operator identity.
- Reason.
- Dry-run option.
- Audit event.
- Replay sau sửa.

---

## 58. Security và Fairness

- Client không được submit price, return, benchmark hoặc Sector Score.
- Mọi người chơi trong Daily Tide dùng cùng Sector metrics.
- Content không thay đổi sau khi Tide mở.
- Basket mapping version phải nằm trong locked rule snapshot.
- Operator repair phải audited.
- Settlement retry không cấp điểm hoặc reward hai lần.
- Public methodology không được làm lộ provider secret.
- Player lineup popularity không ảnh hưởng Sector benchmark.

---

## 59. Implementation Milestones

### Milestone 1 — Content Model

- Sector enum và tables.
- Basket mapping versions.
- Component weights.
- Expected Turbulence policies.
- Content validation.

### Milestone 2 — Basket Metrics

- Observation resolution.
- Component return.
- Coverage và weight renormalization.
- Basket return.
- Normalization.
- Evidence persistence.

### Milestone 3 — Sector Benchmark

- Eligibility.
- Equal-weight benchmark.
- Ranking và percentile.
- Sector Score.
- Checksums và replay.

### Milestone 4 — Settlement Integration

- Keeper result inputs.
- Fleet aggregation.
- Synergy và Modifier rules.
- Result breakdown.
- API model.

### Milestone 5 — Hardening

- Historical simulation.
- Outlier workflow.
- Concurrency tests.
- Admin tools.
- Monitoring.

---

## 60. MVP Acceptance Criteria

Hệ thống Sector hoàn tất khi:

1. Crest, Ember, Current và Harbor tồn tại dưới dạng versioned content.
2. Mọi Keeper tham chiếu đúng một Sector và một basket mapping version.
3. Component weights sum chính xác thành 1.0.
4. Component return dùng cùng Daily Tide window.
5. Missing component được xử lý theo coverage threshold đã publish.
6. Basket return được persist cùng component evidence.
7. Basket được normalize bằng Expected Turbulence và cap hợp lệ.
8. Sector benchmark dùng tất cả eligible unique baskets, không phụ thuộc player lineup.
9. Sector benchmark deterministic, idempotent và replayable.
10. Mỗi Keeper có Sector Relative Performance và Sector Ranking.
11. Sector Score dùng versioned score curve.
12. Fleet Sector Score có thể reconstruct từ Keeper result rows.
13. Settlement bị block khi một Sector bắt buộc chưa READY.
14. Frontend giải thích được benchmark, advantage, rank và contribution.
15. Client không thể override Sector values.
16. Content version mới không sửa historical result.
17. Replay fixtures pass chính xác trong CI.
18. Balance simulation không cho thấy một Sector có structural advantage ngoài ý muốn.

---

## 61. Các quyết định cần chốt thêm

Các tài liệu gameplay/backend hiện tại xác định basket mapping, normalization và Sector-relative comparison, nhưng chưa chốt đầy đủ các nội dung sau:

1. Danh sách market component thật của từng basket.
2. Trọng số component production.
3. Expected Turbulence production dùng static hay rolling.
4. Coverage threshold chính thức của từng Sector.
5. Minimum eligible baskets chính thức.
6. Equal-weight hay configured-weight benchmark.
7. `sector_relative_scale`; spec đề xuất `0.50`.
8. Score blend; spec đề xuất `70% relative + 30% rank`.
9. Fleet aggregation; spec đề xuất average các occupied slots.
10. Cách ghép chính thức Keeper Market Score với Daily Score để tránh double counting.
11. Outlier thresholds.
12. Có công khai component methodology cho người chơi hay không.

Tất cả quyết định phải nằm trong content hoặc rule version, không hard-code.

---

## 62. Canonical Formula Set

```text
Component Return
= (Close Price - Open Price) / Open Price
```

```text
Effective Component Weight
= Target Weight / Sum of Valid Target Weights
```

```text
Basket Return
= Σ(Component Return × Effective Component Weight)
```

```text
Effective Expected Turbulence
= max(Expected Turbulence, Turbulence Floor)
```

```text
Normalized Performance
= clamp(
    Basket Return / Effective Expected Turbulence,
    -Normalization Cap,
    +Normalization Cap
  )
```

```text
Sector Benchmark
= average(
    Normalized Performance của eligible unique baskets trong Sector
  )
```

```text
Sector Relative Performance
= Keeper Normalized Performance - Sector Benchmark
```

```text
Relative Component
= clamp(
    Sector Relative Performance / Sector Relative Scale,
    -1,
    +1
  )
```

```text
Rank Component
= Sector Percentile × 2 - 1
```

```text
Sector Score Normalized
= 0.70 × Relative Component
+ 0.30 × Rank Component
```

```text
Sector Score Points
= round(Sector Score Normalized × 100)
```

```text
Fleet Sector Score
= average(Sector Score Points của occupied Keeper slots)
```

```text
Sector Daily Contribution
= Fleet Sector Score × 20%
```

Mọi constant trong công thức trên là versioned rule value.

---

## 63. Tóm tắt

Bốn Sector không được tính từ số lượng Keeper mà người chơi chọn và cũng không được so sánh trực tiếp bằng raw return.

Mỗi Sector chứa nhiều **unique Keeper baskets**. Mỗi basket được tính từ các market component có trọng số, sau đó normalize bằng Expected Turbulence. Sector Benchmark được tính từ tất cả basket hợp lệ trong Sector. Keeper được đánh giá bằng khoảng cách với benchmark và vị trí xếp hạng trong nhóm.

Cách này bảo đảm:

- Crest có tính ổn định nhưng không mặc định mạnh nhất.
- Ember có upside cao nhưng không được thưởng chỉ vì volatility lớn.
- Current thể hiện đúng vai trò rotation và infrastructure.
- Harbor vẫn có giá trị trong ngày phòng thủ dù raw return nhỏ.
- Tất cả kết quả có thể giải thích, audit và replay.

Câu hỏi gameplay mà Sector system phải trả lời là:

> Keeper này đã hoạt động tốt đến đâu so với các Keeper được thiết kế cho cùng loại Tide?
