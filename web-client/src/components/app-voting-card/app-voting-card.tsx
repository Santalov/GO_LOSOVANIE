import * as React from 'react';
import {ButtonBase, createStyles, makeStyles, Theme} from '@material-ui/core';
import AppInfoLine from '../app-info-line/app-info-line';

export interface AppVotingCardProps {
  hash: string;
  myVotes: number;
}

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    card: {
      width: '100%',
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr) auto',
      gridColumnGap: theme.spacing(1),
      borderBottom: '1px solid',
      borderBottomColor: theme.palette.background.paper,
      textAlign: 'left',
      paddingLeft: theme.spacing(1),
      paddingRight: theme.spacing(1),
      height: '60px',
    }
  })
);

function AppVotingCard({hash, myVotes}: AppVotingCardProps) {
  const classes = useStyles();
  return (
    <ButtonBase
      className={classes.card}
    >
      <AppInfoLine label="Идентификатор голосования">
        {hash}
      </AppInfoLine>
      <AppInfoLine label="Мои голоса">
        {myVotes}
      </AppInfoLine>
    </ButtonBase>
  )
}

export default AppVotingCard
