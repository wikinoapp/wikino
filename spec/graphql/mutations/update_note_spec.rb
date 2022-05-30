# typed: false
# frozen_string_literal: true

describe Mutations::UpdateNote do
  context "success" do
    let!(:user) { create(:user) }
    let!(:note) { create(:note, :with_content, user:, title: "Hello") }
    let!(:variables) { {noteId: NonotoSchema.id_from_object(note), title: note.title, body: "World"} }
    let!(:context) { {viewer: user} }
    let!(:query) do
      <<~GRAPHQL
        mutation($noteId: ID!, $title: String!, $body: String!) {
          updateNote(input: {
            id: $noteId,
            title: $title,
            body: $body
          }) {
            note {
              title

              content {
                body
              }
            }

            errors {
              ... on MutationError {
                message
              }

              ... on DuplicatedNoteError {
                message
                originalNote {
                  title
                }
              }
            }
          }
        }
      GRAPHQL
    end

    it "updates a note" do
      expect(Note.count).to eq(1)

      result = NonotoSchema.execute(query, variables:, context:)

      expect(Note.count).to eq(1)

      expect(result["errors"]).to be_nil
      expect(result.dig("data", "updateNote", "errors")).to eq([])
    end
  end

  context "failure" do
    let!(:query) do
      <<~GRAPHQL
        mutation($noteId: ID!, $title: String!, $body: String!) {
          updateNote(input: {
            id: $noteId,
            title: $title,
            body: $body
          }) {
            note {
              title

              content {
                body
              }
            }

            errors {
              ... on MutationError {
                message
              }

              ... on DuplicatedNoteError {
                message
                originalNote {
                  title
                }
              }
            }
          }
        }
      GRAPHQL
    end

    context "basic mutation error" do
      let!(:user) { create(:user) }
      let!(:original_title) { "Hello" }
      let!(:note) { create(:note, :with_content, user:, title: original_title) }
      let!(:variables) { {noteId: NonotoSchema.id_from_object(note), title: "Updated Hello!", body: "a" * 1_000_001} }
      let!(:context) { {viewer: user} }

      it "returns errors" do
        expect(Note.count).to eq(1)

        result = NonotoSchema.execute(query, variables:, context:)

        expect(Note.count).to eq(1)

        expect(result["errors"]).to be_nil
        expect(result.dig("data", "updateNote", "errors")).to eq([{"message" => "Body is too long (maximum is 1000000 characters)"}])

        note = Note.first
        expect(note.title).to eq(original_title)
      end
    end

    context "duplicated note error" do
      let!(:user) { create(:user) }
      let!(:original_title) { "Hello1" }
      let!(:note) { create(:note, :with_content, user:, title: original_title) }
      let!(:other_note) { create(:note, :with_content, user:, title: "Hello2") }
      let!(:variables) { {noteId: NonotoSchema.id_from_object(note), title: "Hello2", body: "World"} }
      let!(:context) { {viewer: user} }

      it "returns errors" do
        expect(Note.count).to eq(2)

        result = NonotoSchema.execute(query, variables:, context:)

        expect(Note.count).to eq(2)

        expect(result["errors"]).to be_nil
        expect(result.dig("data", "updateNote", "errors")).to eq([{
          "message" => "Title has already existed",
          "originalNote" => {"title" => "Hello2"}
        }])

        expect(Note.find(note.id).title).to eq(original_title)
      end
    end
  end
end
