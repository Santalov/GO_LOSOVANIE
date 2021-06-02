import React from "react";
import {createStyles, makeStyles, Theme} from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    logo: {
      fill: theme.palette.primary.main,
    },
  })
);

function AppLogo() {
  const classes = useStyles();
  return (
    <svg
      width="100%"
      height="100%"
      viewBox="0 0 36 36"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        className={classes.logo}
        d="M31,30H29V27a3,3,0,0,0-3-3H22a3,3,0,0,0-3,3v3H17V27a5,5,0,0,1,5-5h4a5,5,0,0,1,5,5Z"
        transform="translate(0)"/>
      <path
        className={classes.logo}
        d="M24,12a3,3,0,1,1-3,3,3,3,0,0,1,3-3m0-2a5,5,0,1,0,5,5A5,5,0,0,0,24,10Z" transform="translate(0)"/>
      <path
        className={classes.logo}
        d="M15,22H13V19a3,3,0,0,0-3-3H6a3,3,0,0,0-3,3v3H1V19a5,5,0,0,1,5-5h4a5,5,0,0,1,5,5Z"
        transform="translate(0)"/>
      <path
        className={classes.logo}
        d="M8,4A3,3,0,1,1,5,7,3,3,0,0,1,8,4M8,2a5,5,0,1,0,5,5A5,5,0,0,0,8,2Z" transform="translate(0)"/>
    </svg>
  );
}

export default AppLogo;
