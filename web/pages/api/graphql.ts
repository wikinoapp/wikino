import axios from 'axios';

export default async (req: any, res: any) => {
  const { query } = req.body
  const resp = await axios.post('http://api:3000/api/local/graphql', { query })
  res.statusCode = 200
  res.setHeader('Content-Type', 'application/json')
  res.end(JSON.stringify(resp.data))
}
