# frozen_string_literal: true

RSpec.describe Forms::Note, type: :model do
  describe "validations" do
    it "checks body length" do
      body = "a" * 1_000_000

      form = Forms::Note.new(body:)
      form.valid?
      expect(form.errors.of_kind?(:body, :too_long)).to eq(false)

      form = Forms::Note.new(body: body + "a")
      form.valid?
      expect(form.errors.of_kind?(:body, :too_long)).to eq(true)
    end

    it "checks a presence of user" do
      form = Forms::Note.new(user: User.new)
      form.valid?
      expect(form.errors.of_kind?(:user, :blank)).to eq(false)

      form = Forms::Note.new
      form.valid?
      expect(form.errors.of_kind?(:user, :blank)).to eq(true)
    end

    describe "#title_should_be_unique" do
      let!(:user) { create(:user) }
      let!(:note) { create(:note, :with_content, user:, title: "Hello") }

      it "checks unique title" do
        form = Forms::Note.new(user:, title: "Hello")
        form.valid?
        expect(form.errors.of_kind?(:title, :title_should_be_unique)).to eq(true)
      end
    end
  end
end
