export interface AccountStoreLise {
  count: number;
  accounts: AccountStore[];
}

export interface AccountStore {
  id: number;
  name: string;
  spend_pkey: string;
  scan_pkey: string;
  prv: string;
}
