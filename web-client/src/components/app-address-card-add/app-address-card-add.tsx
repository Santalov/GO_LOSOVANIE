import {ButtonBase, createStyles, makeStyles, Typography} from '@material-ui/core';
import AppCard from '../app-card/app-card';

const useStyles = makeStyles((theme) =>
  createStyles({
    container: {
      position: 'relative',
      height: '64px'
    },
    card: {
      visibility: 'hidden',
    },
    addButton: {
      position: 'absolute',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      width: '100%'
    }
  })
);

function AppAddressCardAdd() {
  const classes = useStyles();
  return (
    <AppCard
      className={classes.container}
    >
      <ButtonBase className={classes.addButton}>
        <Typography variant="h5">Добавить</Typography>
      </ButtonBase>
    </AppCard>
  )
}

export default AppAddressCardAdd
