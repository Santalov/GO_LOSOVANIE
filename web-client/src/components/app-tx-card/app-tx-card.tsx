import React from 'react';
import {ButtonBase, createStyles, makeStyles, Theme, Typography} from '@material-ui/core';
import AppInfoLine from '../app-info-line/app-info-line';
import {ArrowDownward, ArrowUpward, HowToVote} from '@material-ui/icons';
import classNames from 'classnames';

export interface AppTxCardProps {
  type: 'in' | 'out' | 'voteIn' | 'voteOut';
  name: string; // receiver or sender name or address
  value: number; // formatted number
}

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    card: {
      display: 'grid',
      gridTemplateColumns: 'auto minmax(0, 1fr) auto',
      gridColumnGap: theme.spacing(1),
      borderBottom: '1px solid',
      borderBottomColor: theme.palette.background.paper,
      textAlign: 'left',
      paddingLeft: theme.spacing(1),
      paddingRight: theme.spacing(1),
      height: '60px',
    },
    icon: {
      color: theme.palette.text.secondary,
    },
    name: {
      width: '100%',
      overflow: 'hidden'
    },
    valueIn: {
      color: theme.palette.secondary.main,
    },
    valueOut: {
      color: theme.palette.text.primary,
    }
  })
);

function AppTxCard({type, name, value}: AppTxCardProps) {
  const classes = useStyles();
  let icon;
  switch (type) {
    case 'in':
      icon = (<ArrowDownward color="inherit"/>);
      break;
    case 'out':
      icon = (<ArrowUpward color="inherit"/>);
      break;
    case 'voteIn':
    case 'voteOut':
      icon = (<HowToVote color="inherit"/>);
      break;
  }
  return (
    <ButtonBase
      className={classes.card}
    >
      <div className={classes.icon}>
        {icon}
      </div>
      <AppInfoLine
        label={type === 'in' || type === 'voteIn' ? 'От' : 'Кому'}
      >
        {name}
      </AppInfoLine>
      <Typography
        variant="h5"
        className={classNames({
          [classes.valueIn]: type === 'in' || type === 'voteIn',
          [classes.valueOut]: type === 'out' || type === 'voteOut',
        })}
      >
        {value}
      </Typography>
    </ButtonBase>
  )
}

export default AppTxCard
