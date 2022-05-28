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
              message
            }
          }
        }
      GRAPHQL
    end

    it "creates a note" do
      expect(Note.count).to eq(0)

      result = NonotoSchema.execute(query, context:)

      expect(Note.count).to eq(1)
      note = user.notes.first

      expect(result["errors"]).to be_nil
      expect(result.dig("data", "createNote", "note", "id")).to eq(NonotoSchema.id_from_object(note))
      expect(result.dig("data", "createNote", "errors")).to eq([])
    end
  end

  context "failure" do
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
              message
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
      expect(result.dig("data", "createNote", "errors")).to eq([{"message" => "Title has already existed"}])
    end
  end
end
