# typed: false
# frozen_string_literal: true

RSpec.describe NoteCreatingForm, type: :model do
  describe "validations" do
    it "checks body length" do
      body = "a" * 1_000_000

      form = NoteCreatingForm.new(body:)
      form.valid?
      expect(form.errors.of_kind?(:body, :too_long)).to eq(false)

      form = NoteCreatingForm.new(body: body + "a")
      form.valid?
      expect(form.errors.of_kind?(:body, :too_long)).to eq(true)
    end

    it "checks a presence of user" do
      form = NoteCreatingForm.new(user: User.new)
      form.valid?
      expect(form.errors.of_kind?(:user, :blank)).to eq(false)

      form = NoteCreatingForm.new
      form.valid?
      expect(form.errors.of_kind?(:user, :blank)).to eq(true)
    end

    describe "#title_should_be_unique" do
      let!(:user) { create(:user) }
      let!(:note) { create(:note, :with_content, user:, title: "Hello") }

      it "checks unique title" do
        form = NoteCreatingForm.new(user:, title: "Hello")
        form.valid?
        expect(form.errors.of_kind?(:title, :title_should_be_unique)).to eq(true)
      end
    end
  end
end
