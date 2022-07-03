# typed: false
# frozen_string_literal: true

RSpec.describe CreateNoteService, type: :model do
  context "when the service succeeds" do
    let!(:user) { create(:user) }

    it "creates a note" do
      form = NoteCreatingForm.new(user:, title: "Hello", body: "World")
      service = CreateNoteService.new(form:)

      expect(Note.count).to eq(0)
      expect(NoteContent.count).to eq(0)

      result = service.call

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.note.user).to eq(user)
      expect(result.note.title).to eq("Hello")
      expect(result.note.body).to eq("World")
      expect(result.errors).to eq([])
    end
  end

  context "when the service fails" do
    let!(:user) { create(:user) }

    before do
      create(:note, :with_content, user:, title: "Hello")
    end

    it "returns errors" do
      form = NoteCreatingForm.new(user:, title: "Hello", body: "World")
      service = CreateNoteService.new(form:)

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      result = service.call

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.note).to be_nil
      expect(result.errors.map(&:message)).to eq(["Title has already existed"])
    end
  end
end
