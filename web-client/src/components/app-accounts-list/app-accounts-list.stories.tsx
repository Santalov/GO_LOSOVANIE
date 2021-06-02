import AppAccountsList from './app-accounts-list';
import React from 'react';
import {mockAccounts} from './mock-accounts';

export default {
  title: 'AppAccountsList',
  component: AppAccountsList
}

const accSelected = (id: number) => console.log('account selected', id);
const accountAddClicked = () => console.log('add account');

export const Default = () => (
  <AppAccountsList
    accounts={mockAccounts}
    selectedAccountChanged={accSelected}
    accountAddClicked={accountAddClicked}
  />
);
