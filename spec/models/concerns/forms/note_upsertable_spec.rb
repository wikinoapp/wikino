# typed: false
# frozen_string_literal: true

RSpec.describe Forms::NoteUpsertable, type: :model do
  class TestNoteUpsertableForm < Forms::ApplicationForm
    extend T::Sig
    include Forms::NoteUpsertable
  end

  describe "validations" do
    describe "body length" do
      let!(:user) { create(:user) }

      it "checks body length" do
        body = "a" * 1_000_000

        form = TestNoteUpsertableForm.new(user:, body:)
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to eq(false)

        form = TestNoteUpsertableForm.new(user:, body: body + "a")
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to eq(true)
      end
    end
  end
end
