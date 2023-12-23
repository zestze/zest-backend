'use client'

import { useState } from 'react';
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Toolbar from '@mui/material/Toolbar';
import Stack from '@mui/material/Stack';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import CardHeader from '@mui/material/CardHeader';
import { CardActionArea } from '@mui/material';
import Typography from '@mui/material/Typography';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';

import { usePosts, Post } from './api';

const drawerWidth = 240;

interface Props {
  posts: Post[]
}

// TODO(zeke): turn this into a Grid?
const PostsPage = ({ posts }: Props) => {
  return <Stack spacing={2}>{posts.map((p: Post) => {
    const redirect = `https://www.metacritic.com${p.href}`;
    return <Card >
      <CardActionArea href={redirect}>
        <CardHeader title={p.title} subheader={p.score}>
        </CardHeader>
        <CardContent>
          <Typography variant="body1" color="text.secondary" gutterBottom paragraph>
            {p.description}
          </Typography>
        </CardContent>
      </CardActionArea>
    </Card>
  })}</Stack>
};

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
  },
});

interface YearFormProps {
  id: string;
  name: string;
  value: number;
  setter: any;
}

const YearForm = ({ id, name, value, setter }: YearFormProps) => {
  const yearItems = Array(2021, 2022, 2023).map((y: number) =>
    <MenuItem value={`${y}`}>{y}</MenuItem>
  );
  return (
    <FormControl>
      <InputLabel id={`${id}-select-label`}>{name}</InputLabel>
      <Select
        labelId={`${id}-select-label`}
        id={`${id}-select`}
        value={`${value}`}
        label={`${name}`}
        onChange={(event: SelectChangeEvent) => {
          setter(parseInt(event.target.value as string))
        }}
      >
        {yearItems}
      </Select>
    </FormControl >
  );
};

export default function Home() {
  const [medium, setMedium] = useState<string>("switch");
  const [startYear, setStartYear] = useState<number>(2021);
  const [endYear, setEndYear] = useState<number>(2023);
  const [posts, isLoading] = usePosts(medium, startYear, 2023);

  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />
      <Box sx={{ flexGrow: 1, pt: 1 }}>
        <AppBar
          position="static"
        >
          <Toolbar>
            <Stack
              direction="row"
              spacing={2}
              alignItems="center"
              justifyContent="flex-start"
            >
              <FormControl>
                <InputLabel id="medium-select-label">Medium</InputLabel>
                <Select
                  labelId="medium-select-label"
                  id="medium-select"
                  value={medium}
                  label="Medium"
                  onChange={(event: SelectChangeEvent) => {
                    setMedium(event.target.value as string);
                  }}
                >
                  <MenuItem value="tv">tv</MenuItem>
                  <MenuItem value="pc">pc</MenuItem>
                  <MenuItem value="switch">switch</MenuItem>
                  <MenuItem value="movie">movie</MenuItem>
                </Select>
              </FormControl>
              <YearForm
                id="startyear"
                name="Start Year"
                value={startYear}
                setter={setStartYear}
              />
              <YearForm
                id="endyear"
                name="End Year"
                value={endYear}
                setter={setEndYear}
              />
              <Typography variant="h5" noWrap component="div">
                BetaCritic
              </Typography>
            </Stack>
          </Toolbar>
        </AppBar>
      </Box>
      <Box
        component="main"
        sx={{ flexGrow: 1, px: 3, width: { sm: `calc(100% - ${drawerWidth}px)` } }}
      >
        <Toolbar />
        {!isLoading ? (<PostsPage posts={posts} />) : <h3>loading</h3>}
      </Box>
    </ThemeProvider >
  );
}
