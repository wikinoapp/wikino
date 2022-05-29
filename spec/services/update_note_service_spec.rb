# typed: false
# frozen_string_literal: true

RSpec.describe UpdateNoteService, type: :model do
  context "success" do
    let!(:user) { create(:user) }
    let!(:note) { create(:note, :with_content, user:, title: "Hello") }

    it "updates the note" do
      form = NoteUpdatingForm.new(user:, note:, title: "Hello", body: "World")
      service = UpdateNoteService.new(form:)

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      result = service.call

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.errors).to eq([])
      expect(result.note.body).to eq("World")
    end
  end

  context "failure" do
    let!(:user) { create(:user) }
    let!(:original_note) { create(:note, :with_content, user: user, title: "Original Hello") }
    let!(:note) { create(:note, :with_content, user: user, title: "Hello") }

    it "returns errors" do
      form = NoteUpdatingForm.new(user:, note:, title: "Original Hello", body: "a" * 1_000_001)
      service = UpdateNoteService.new(form:)

      expect(Note.count).to eq(2)
      expect(NoteContent.count).to eq(2)

      result = service.call

      expect(Note.count).to eq(2)
      expect(NoteContent.count).to eq(2)

      expect(result.errors.map(&:code)).to eq(["INVALID_ERROR", "DUPLICATED_NOTE_ERROR"])
      expect(result.errors.map(&:message)).to eq(["Body is too long (maximum is 1000000 characters)", "Title has already existed"])
      expect(result.errors.map(&:original_note)).to eq([nil, original_note])
    end
  end
end
