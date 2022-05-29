# typed: false
# frozen_string_literal: true

RSpec.describe NoteDestroyingForm, type: :model do
  describe "validations" do
    describe "#only_own_note_could_be_destroyed" do
      let!(:user) { create(:user) }
      let!(:other_user) { create(:user) }
      let!(:note) { create(:note, :with_content, user: other_user, title: "Hello") }

      it "checks a user who creates the note" do
        form = NoteDestroyingForm.new(user:, note:)
        form.valid?
        expect(form.errors.of_kind?(:note, :only_own_note_could_be_destroyed)).to eq(true)

        form = NoteDestroyingForm.new(user: other_user, note:)
        form.valid?
        expect(form.errors.of_kind?(:note, :only_own_note_could_be_destroyed)).to eq(false)
      end
    end
  end
end
