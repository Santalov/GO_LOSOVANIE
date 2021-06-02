import React, {useState} from 'react';
import {Account} from '../../models/account';
import AppAddressCard from '../app-address-card/app-address-card';
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import {dashboard} from '../../theme/app-theme-constants';
import AppAddressCardAdd from '../app-address-card-add/app-address-card-add';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    card: {
      marginBottom: theme.spacing(dashboard.interCardPadding)
    }
  })
);

export interface AppAccountsListProps {
  accounts: Account[]
  selectedAccountChanged: (accountId: number) => void;
  accountAddClicked: () => void;
}

function AppAccountsList({accounts, selectedAccountChanged, accountAddClicked}: AppAccountsListProps) {
  const [currentAcc, setCurrentAcc] = useState(accounts[0]?.id || 0);
  const accClick = (id: number) => {
    setCurrentAcc(id);
    selectedAccountChanged(id);
  };
  const classes = useStyles();
  return (
    <>
      {
        accounts.map(acc => (
          <div
            onClick={() => accClick(acc.id)}
            className={classes.card}
            key={acc.id}
          >
            <AppAddressCard
              name={acc.name}
              address={acc.spend_pkey}
              coins={acc.coins}
              votes={acc.votes}
              active={acc.id === currentAcc}
            />
          </div>
        ))
      }
      <div
        onClick={accountAddClicked}
        className={classes.card}
      >
        <AppAddressCardAdd/>
      </div>
    </>
  )
}

export default AppAccountsList
