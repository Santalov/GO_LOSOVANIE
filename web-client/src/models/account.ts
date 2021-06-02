export interface Account {
  id: number;
  spend_pkey: string;
  scan_pkey: string;
  prv: string;
  name: string;
  votes: number;
  coins: number;
  utxo: any[];
}
