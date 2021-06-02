import {createStyles, makeStyles, Theme} from '@material-ui/core/styles';
import AppCard from '../app-card/app-card';
import classNames from 'classnames';
import {IconButton, Typography} from '@material-ui/core';
import {ArrowDownward, ArrowUpward, FileCopy, History, HowToVote} from '@material-ui/icons';
import {SyntheticEvent} from 'react';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      borderLeft: '2px solid',
      borderLeftColor: 'transparent',
      '&:hover': {
        backgroundColor: 'rgba(0, 0, 0, 0.1)',
        cursor: 'pointer',
      }
    },
    rootActive: {
      borderLeftColor: theme.palette.primary.main
    },
    header: {
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr) auto',
      gridTemplateRows: 'auto auto',
      gridRowGap: theme.spacing(0.25),
      gridTemplateAreas: `
        'name coins'
        'name votes'
      `,
      padding: theme.spacing(1.5),
    },
    headline: {
      display: 'flex',
      flexDirection: 'column',
      flexWrap: 'nowrap',
      justifyContent: 'center',
      gridArea: 'name',
      width: '100%',
      overflow: 'hidden',
    },
    headlineItem: {
      width: '100%',
      whiteSpace: 'nowrap',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
    },
    name: {
      color: theme.palette.text.primary,
    },
    address: {
      color: theme.palette.text.secondary,
      width: '100%',
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr) auto',
    },
    values: {
      display: 'flex',
      justifyContent: 'flex-end',
    },
    coins: {
      color: theme.palette.text.primary,
    },
    votes: {
      color: theme.palette.primary.main,
    },
    actions: {
      paddingLeft: theme.spacing(4),
      paddingRight: theme.spacing(1.5),
      paddingBottom: theme.spacing(2),
      display: 'flex',
      justifyContent: 'flex-end'
    },
    actionButton: {
      marginLeft: theme.spacing(1),
    }
  })
);

export interface AppAddressCardProps {
  name: string;
  address: string;
  coins: number;
  votes: number;
  active: boolean;
}

function AppAddressCard(
  {name, address, coins, votes, active}: AppAddressCardProps
) {
  const classes = useStyles();
  const copyAddress = (e: SyntheticEvent) => {
    e.preventDefault();
    e.stopPropagation();
    navigator
      .clipboard
      .writeText(address)
      .then(() => {
          console.log('copied', address)
        }
      );
  };
  return (
    <AppCard className={classNames(classes.root, {[classes.rootActive]: active})}>
      <div className={classes.header}>
        <div className={classes.headline}>
          <Typography
            className={classNames(classes.headlineItem, classes.name)}
            variant="h4"
          >
            {name}
          </Typography>
          <div className={classes.address}>
            <Typography
              variant="subtitle1"
              className={classes.headlineItem}
            >
              {address}
            </Typography>
            <IconButton
              aria-label="delete"
              size="small"
              className="clipboard-button"
              onClick={copyAddress}
            >
              <FileCopy fontSize="inherit"/>
            </IconButton>
          </div>
        </div>
        <Typography
          className={classNames(classes.values, classes.coins)}
          variant="h5"
        >
          {coins} м
        </Typography>
        <Typography
          className={classNames(classes.values, classes.votes)}
          variant="h5"
        >
          {votes} г
        </Typography>
      </div>
      {
        active &&
        <div className={classes.actions}>
          <IconButton className={classes.actionButton}>
            <ArrowUpward/>
          </IconButton>
          <IconButton className={classes.actionButton}>
            <ArrowDownward/>
          </IconButton>
          <IconButton className={classes.actionButton}>
            <History/>
          </IconButton>
          <IconButton className={classes.actionButton}>
            <HowToVote/>
          </IconButton>
        </div>
      }
    </AppCard>
  )
}

export default AppAddressCard
