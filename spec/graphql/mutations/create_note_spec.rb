# typed: false
# frozen_string_literal: true

describe Mutations::CreateNote do
  context "success" do
    let!(:user) { create(:user) }
    let!(:context) { {viewer: user} }
    let!(:query) do
      <<~GRAPHQL
        mutation {
          createNote(input: {
            title: "Hello",
            body: "World"
          }) {
            note {
              id
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

    it "creates a note" do
      expect(Note.count).to eq(0)

      result = NonotoSchema.execute(query, context:)

      expect(result["errors"]).to be_nil
      expect(result.dig("data", "createNote", "errors")).to eq([])

      expect(Note.count).to eq(1)
      note = user.notes.first

      expect(result.dig("data", "createNote", "note", "id")).to eq(NonotoSchema.id_from_object(note))
    end
  end

  context "failure" do
    context "basic mutation error" do
      let!(:user) { create(:user) }
      let!(:context) { {viewer: user} }
      let!(:query) do
        <<~GRAPHQL
          mutation($body: String!) {
            createNote(input: {
              title: "Hello",
              body: $body
            }) {
              note {
                id
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

      it "returns errors" do
        expect(Note.count).to eq(0)

        result = NonotoSchema.execute(query, variables: {body: "a" * 1_000_001}, context:)

        expect(Note.count).to eq(0)

        expect(result["errors"]).to be_nil
        expect(result.dig("data", "createNote", "note")).to be_nil
        expect(result.dig("data", "createNote", "errors")).to eq([{
          "message" => "Body is too long (maximum is 1000000 characters)"
        }])
      end
    end

    context "duplicated note error" do
      let!(:user) { create(:user) }
      let!(:context) { {viewer: user} }
      let!(:query) do
        <<~GRAPHQL
          mutation($title: String!) {
            createNote(input: {
              title: $title,
              body: "World"
            }) {
              note {
                id
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

      before do
        create(:note, :with_content, user:, title: "Hello")
      end

      it "returns errors" do
        expect(Note.count).to eq(1)

        result = NonotoSchema.execute(query, variables: {title: "Hello"}, context:) # duplicated title

        expect(Note.count).to eq(1)

        expect(result["errors"]).to be_nil
        expect(result.dig("data", "createNote", "note")).to be_nil
        expect(result.dig("data", "createNote", "errors")).to eq([{
          "message" => "Title has already existed",
          "originalNote" => {"title" => "Hello"}
        }])
      end
    end
  end
end
