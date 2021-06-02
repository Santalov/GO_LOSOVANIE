import {Account} from '../../models/account';

export const mockAccounts: Account[] = [
  {
    id: 1,
    name: 'Poseidon',
    scan_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    spend_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    prv: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    coins: 1000,
    votes: 15,
    utxo: []
  },
  {
    id: 2,
    name: 'Gvidon',
    scan_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    spend_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    prv: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    coins: 111,
    votes: 1337,
    utxo: []
  },
  {
    id: 3,
    name: 'Megaladon',
    scan_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    spend_pkey: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    prv: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
    coins: 1000000,
    votes: 13370000,
    utxo: []
  }
];
