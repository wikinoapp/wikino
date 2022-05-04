# frozen_string_literal: true

describe "POST /internal/graphql", type: :request do
  let!(:user) { create(:user) }
  let!(:access_token) { create(:access_token, user: user) }
  let!(:note) { create(:note, user: user) }
  let!(:query) do
    <<~GRAPHQL
      query($noteId: String!) {
        viewer {
          note(databaseId: $noteId) {
            databaseId
            title
          }
        }
      }
    GRAPHQL
  end
  let!(:headers) { {"Authorization" => "Bearer #{access_token.token}"} }

  it "should response" do
    post "/internal/graphql", params: {variables: {noteId: note.id}, query: query}, headers: headers

    expect(response.status).to eq(200)
    expect(JSON.parse(response.body)).to include({
      data: {
        viewer: {
          note: {
            databaseId: note.id,
            title: note.title,
          }
        }
      }
    }.deep_stringify_keys)
  end
end
