# typed: false
# frozen_string_literal: true

RSpec.describe NoteInputtable, type: :model do
  describe "validations" do
    describe "body length" do
      let!(:user) { create(:user) }

      before do
        stub_const("TestNoteInputtableForm", Class.new(ApplicationForm) do |klass|
          klass.include NoteInputtable

          private

          def user_notes
            user.notes
          end
        end)
      end

      it "checks body length" do
        body = "a" * 1_000_000

        form = TestNoteInputtableForm.new(user:, body:)
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to be(false)

        form = TestNoteInputtableForm.new(user:, body: body + "a")
        form.valid?
        expect(form.errors.of_kind?(:body, :too_long)).to be(true)
      end
    end
  end
end
