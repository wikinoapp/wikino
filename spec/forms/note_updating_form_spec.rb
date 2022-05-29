# typed: false
# frozen_string_literal: true

RSpec.describe NoteUpdatingForm do
  describe "validations" do
    describe "#title_should_be_unique" do
      context "same note" do
        let!(:user) { create(:user) }
        let!(:note) { create(:note, :with_content, user:, title: "Hello") }

        it "returns no error" do
          form = NoteUpdatingForm.new(user:, note:, title: "Hello")
          form.valid?
          expect(form.errors.of_kind?(:title, :title_should_be_unique)).to eq(false)
        end
      end

      context "different note" do
        let!(:user) { create(:user) }
        let!(:note) { create(:note, :with_content, user:, title: "Hello") }

        before do
          create(:note, :with_content, user:, title: "Original Hello")
        end

        it "returns error" do
          form = NoteUpdatingForm.new(user:, note:, title: "Original Hello")
          form.valid?
          expect(form.errors.of_kind?(:title, :title_should_be_unique)).to eq(true)
        end
      end
    end
  end
end
