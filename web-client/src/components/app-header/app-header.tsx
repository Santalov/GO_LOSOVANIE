import React, {PropsWithChildren} from "react";
import AppLogoGroup from "../app-logo-group/app-logo-group";
import AppCard from '../app-card/app-card';
import {createStyles, makeStyles} from '@material-ui/core';

const useStyles = makeStyles((theme) =>
  createStyles({
    header: {
      margin: 0,
      padding: 0,
    },
    headCard: {
      display: "grid",
      gridTemplateColumns: "auto 1fr",
      gridColumnGap: theme.spacing(1),
      marginBottom: theme.spacing(3),
      height: '70px',
      alignItems: 'center',
      paddingLeft: theme.spacing(10),
    }
  })
);

function AppHeader(props: PropsWithChildren<{}>) {
  const classes = useStyles();
  return (
    <header className={classes.header}>
      <AppCard className={classes.headCard} sharp={true}>
        <AppLogoGroup/>
        <div>{props.children}</div>
      </AppCard>
    </header>
  );
}

export default AppHeader;
