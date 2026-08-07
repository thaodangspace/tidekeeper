export type Affinity = 'crest' | 'harbor' | 'echo' | 'current' | 'ember' | 'bloom' | 'unknown';

export interface Keeper {
  id: string;
  name: string;
  title: string;
  role: string;
  passive: string;
  affinity: Affinity;
  image?: string;
}

export interface Offer extends Keeper {
  cost: number;
}

export const fleet: Keeper[] = [
  {
    id: 'crest-guardian',
    name: 'Crest Guardian',
    title: 'Crest · Tiên phong',
    role: 'Tiên phong',
    passive: 'Nhận Giáp khi đội kiếm được Giáp.',
    affinity: 'crest',
    image: '/art/crest-guardian.webp'
  },
  {
    id: 'harbor-warden',
    name: 'Harbor Warden',
    title: 'Harbor · Hộ vệ',
    role: 'Hộ vệ',
    passive: 'Giảm Áp lực khi thắng ngày.',
    affinity: 'harbor',
    image: '/art/harbor-warden.webp'
  },
  {
    id: 'echo-seer',
    name: 'Echo Seer',
    title: 'Echo · Tiên tri',
    role: 'Tiên tri',
    passive: 'Tiết lộ 1 hiệu ứng Thủy triều tương lai.',
    affinity: 'echo',
    image: '/art/echo-seer.webp'
  },
  {
    id: 'current-weaver',
    name: 'Current Weaver',
    title: 'Current · Hỗ trợ',
    role: 'Hỗ trợ',
    passive: 'Nhận Dòng chảy khi hỗ trợ đồng minh.',
    affinity: 'current',
    image: '/art/current-weaver.webp'
  }
];

export const initialOffers: Offer[] = [
  {
    id: 'ember-trickster', name: 'Ember Trickster', title: 'Ember · Lửa gạt', role: 'Lửa gạt',
    passive: 'Kích hoạt Ember II', affinity: 'ember', image: '/art/ember-trickster.webp', cost: 5
  },
  {
    id: 'harbor-scout', name: 'Harbor Scout', title: 'Harbor · Trinh sát', role: 'Trinh sát',
    passive: 'Tăng phòng thủ.', affinity: 'harbor', image: '/art/harbor-scout.webp', cost: 3
  },
  {
    id: 'bloom-tender', name: 'Bloom Tender', title: 'Bloom · Hỗ trợ', role: 'Hỗ trợ',
    passive: 'Tăng Tăng trưởng mỗi ngày.', affinity: 'bloom', image: '/art/bloom-tender.webp', cost: 4
  }
];
