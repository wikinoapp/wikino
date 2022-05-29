# typed: false
# frozen_string_literal: true

RSpec.describe DestroyNoteService, type: :model do
  context "success" do
    let!(:user) { create(:user) }
    let!(:note) { create(:note, :with_content, user:, title: "Hello") }

    it "destroys the note" do
      form = NoteDestroyingForm.new(user:, note:)
      service = DestroyNoteService.new(form:)

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      result = service.call

      expect(Note.count).to eq(0)
      expect(NoteContent.count).to eq(0)

      expect(result.errors).to eq([])
    end
  end

  context "failure" do
    let!(:user) { create(:user) }
    let!(:other_user) { create(:user) }
    let!(:note) { create(:note, :with_content, user: other_user, title: "Hello") }

    it "returns errors" do
      form = NoteDestroyingForm.new(user:, note:)
      service = DestroyNoteService.new(form:)

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      result = service.call

      expect(Note.count).to eq(1)
      expect(NoteContent.count).to eq(1)

      expect(result.errors.map(&:message)).to eq(["Note can't destroy"])
    end
  end
end
