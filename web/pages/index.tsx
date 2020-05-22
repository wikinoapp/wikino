import withApollo from '../lib/with-apollo'
import { useViewerQuery } from '../lib/graphql-client-api'

const Index = () => {
  const viewerQuery = useViewerQuery()

  if (viewerQuery.data) {
    return (
      <div>
        Hello, {viewerQuery.data.testField}
      </div>
    )
  }

  return <div>...</div>
}

export default withApollo(Index)
