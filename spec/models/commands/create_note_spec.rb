# typed: false
# frozen_string_literal: true

RSpec.describe Commands::CreateNote, type: :model do
  context "success" do
    let!(:user) { create(:user) }

    it "creates a note" do
      form = Forms::Note.new(user:, title: "Hello", body: "World")
      command = Commands::CreateNote.new(form:)

      expect(Note.count).to eq(0)
      expect(NoteContent.count).to eq(0)

      result = command.run

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.note.user).to eq(user)
      expect(result.note.title).to eq("Hello")
      expect(result.note.body).to eq("World")
      expect(result.errors).to eq([])
    end
  end

  context "failure" do
    let!(:user) { create(:user) }

    before do
      create(:note, :with_content, user:, title: "Hello")
    end

    it "returns errors" do
      form = Forms::Note.new(user:, title: "Hello", body: "World")
      command = Commands::CreateNote.new(form:)

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      result = command.run

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.note).to be_nil
      expect(result.errors.map(&:message)).to eq(["Title has already existed"])
    end
  end
end
