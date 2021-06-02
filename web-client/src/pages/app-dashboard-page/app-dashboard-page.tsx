import React from 'react';
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import {dashboard} from '../../theme/app-theme-constants';
import AppHeader from '../../components/app-header/app-header';
import AppAccountsList from '../../components/app-accounts-list/app-accounts-list';
import {mockAccounts} from '../../components/app-accounts-list/mock-accounts';
import AppSendPage from '../app-send-page/app-send-page';
import AppReceivePage from '../app-receive-page/app-receive-page';
import AppVotingPage from '../app-voting-page/app-voting-page';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    content: {
      paddingLeft: theme.spacing(dashboard.sidePadding),
      paddingRight: theme.spacing(dashboard.sidePadding),
      display: 'grid',
      gridTemplateColumns: 'minmax(400px, 1fr) 2.2fr',
      gridColumnGap: theme.spacing(dashboard.interCardPadding * 2)
    }
  })
);


function AppDashboardPage() {
  const classes = useStyles();
  const callback = () => {
  };
  return (
    <>
      <AppHeader/>
      <div className={classes.content}>
        <div>
          <AppAccountsList
            accounts={mockAccounts}
            selectedAccountChanged={callback}
            accountAddClicked={callback}
          />
        </div>
        <div>
          {/*<AppSendPage/>*/}
          {/*<AppReceivePage/>*/}
          <AppVotingPage/>
        </div>
      </div>
    </>
  )
}

export default AppDashboardPage
