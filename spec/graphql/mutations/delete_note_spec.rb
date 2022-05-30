# typed: false
# frozen_string_literal: true

describe Mutations::DeleteNote do
  context "success" do
    let!(:user) { create(:user) }
    let!(:note) { create(:note, :with_content, user:, title: "Hello") }
    let!(:variables) { {noteId: NonotoSchema.id_from_object(note)} }
    let!(:context) { {viewer: user} }
    let!(:query) do
      <<~GRAPHQL
        mutation($noteId: ID!) {
          deleteNote(input: {
            id: $noteId
          }) {
            errors {
              ... on MutationError {
                message
              }
            }
          }
        }
      GRAPHQL
    end

    it "deletes a note" do
      expect(Note.count).to eq(1)

      result = NonotoSchema.execute(query, variables:, context:)

      expect(Note.count).to eq(0)

      expect(result["errors"]).to be_nil
      expect(result.dig("data", "deleteNote", "errors")).to eq([])
    end
  end

  context "failure" do
    let!(:user) { create(:user) }
    let!(:other_user) { create(:user) }
    let!(:note) { create(:note, :with_content, user: other_user, title: "Hello") }
    let!(:variables) { {noteId: NonotoSchema.id_from_object(note)} }
    let!(:context) { {viewer: user} }
    let!(:query) do
      <<~GRAPHQL
        mutation($noteId: ID!) {
          deleteNote(input: {
            id: $noteId
          }) {
            errors {
              ... on MutationError {
                message
              }
            }
          }
        }
      GRAPHQL
    end

    it "returns errors" do
      expect(Note.count).to eq(1)

      result = NonotoSchema.execute(query, variables:, context:)

      expect(Note.count).to eq(1)

      expect(result["errors"]).to be_nil
      expect(result.dig("data", "deleteNote", "errors")).to eq([{"message" => "Note can't be blank"}])
    end
  end
end
