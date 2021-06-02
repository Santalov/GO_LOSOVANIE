import React, {PropsWithChildren} from 'react';
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import Typography from '@material-ui/core/Typography';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    headline: {
      marginBottom: theme.spacing(3)
    }
  })
);

function AppHeadline({children}: PropsWithChildren<{}>) {
  const classes = useStyles();
  return (
    <Typography
      variant="h3"
      className={classes.headline}
    >
      {children}
    </Typography>
  )
}

export default AppHeadline
