# typed: false
# frozen_string_literal: true

describe "POST /internal/graphql", type: :request do
  let!(:user) { create(:user) }
  let!(:id_token) { "xxxxx" }
  let!(:note) { create(:note, user:) }
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
  let!(:headers) { {"Authorization" => "Bearer #{id_token}"} }

  before do
    create(:note_content, user:, note:)

    allow(JsonWebToken).to receive(:decode_id_token).with(id_token).and_return([{"sub" => user.auth0_user_id}, {}])
  end

  it "responses" do
    post("/internal/graphql", params: {variables: {noteId: note.id}, query:}, headers:)

    expect(response.status).to eq(200)
    expect(JSON.parse(response.body)).to include({
      data: {
        viewer: {
          note: {
            databaseId: note.id,
            title: note.title
          }
        }
      }
    }.deep_stringify_keys)
  end
end
