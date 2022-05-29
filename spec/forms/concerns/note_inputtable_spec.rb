# typed: false
# frozen_string_literal: true

RSpec.describe NoteInputtable, type: :model do
  class TestNoteInputtableForm < ApplicationForm
    include NoteInputtable

    private

    def user_notes
      user.notes
    end
  end

  describe "validations" do
    describe "body length" do
      let!(:user) { create(:user) }

      it "checks body length" do
        body = "a" * 1_000_000

        form = TestNoteInputtableForm.new(user:, body:)
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to eq(false)

        form = TestNoteInputtableForm.new(user:, body: body + "a")
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to eq(true)
      end
    end
  end
end
