# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe Pages::SearchForm, type: :form do
  around do |example|
    I18n.with_locale(:ja) do
      example.run
    end
  end

  describe "バリデーション" do
    it "有効なキーワードの場合、バリデーションが通ること" do
      form = Pages::SearchForm.new(q: "テストキーワード")
      expect(form).to be_valid
    end

    it "空文字列の場合、バリデーションが通ること" do
      form = Pages::SearchForm.new(q: "")
      expect(form).to be_valid
    end

    it "nilの場合、バリデーションが通ること" do
      form = Pages::SearchForm.new(q: nil)
      expect(form).to be_valid
    end

    it "1文字の場合、バリデーションエラーになること" do
      form = Pages::SearchForm.new(q: "a")
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("は2文字以上で入力してください")
    end

    it "101文字以上の場合、バリデーションエラーになること" do
      form = Pages::SearchForm.new(q: "a" * 101)
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("は100文字以内で入力してください")
    end

    it "不正な文字(<や>)が含まれている場合、バリデーションエラーになること" do
      form = Pages::SearchForm.new(q: "test<script>")
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("に不正な文字が含まれています")
    end
  end

  describe "#query_present?" do
    it "キーワードが存在する場合、trueを返すこと" do
      form = Pages::SearchForm.new(q: "テスト")
      expect(form.query_present?).to be true
    end

    it "キーワードが空文字列の場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: "")
      expect(form.query_present?).to be false
    end

    it "キーワードがnilの場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: nil)
      expect(form.query_present?).to be false
    end
  end

  describe "#searchable?" do
    it "有効でキーワードが存在する場合、trueを返すこと" do
      form = Pages::SearchForm.new(q: "テスト")
      expect(form.searchable?).to be true
    end

    it "無効な場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: "a")
      expect(form.searchable?).to be false
    end

    it "キーワードが存在しない場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: "")
      expect(form.searchable?).to be false
    end
  end

  describe "#space_identifiers" do
    it "space:指定子が1つの場合、スペース識別子を抽出すること" do
      form = Pages::SearchForm.new(q: "space:my-space テストキーワード")
      expect(form.space_identifiers).to eq(["my-space"])
    end

    it "space:指定子が複数の場合、全てのスペース識別子を抽出すること" do
      form = Pages::SearchForm.new(q: "space:space1 space:space2 テストキーワード")
      expect(form.space_identifiers).to eq(["space1", "space2"])
    end

    it "space:指定子がない場合、空配列を返すこと" do
      form = Pages::SearchForm.new(q: "テストキーワード")
      expect(form.space_identifiers).to eq([])
    end

    it "キーワードがnilの場合、空配列を返すこと" do
      form = Pages::SearchForm.new(q: nil)
      expect(form.space_identifiers).to eq([])
    end
  end

  describe "#keyword_without_space_filters" do
    it "space:指定子を除いたキーワードを返すこと" do
      form = Pages::SearchForm.new(q: "space:my-space テストキーワード")
      expect(form.keyword_without_space_filters).to eq("テストキーワード")
    end

    it "複数のspace:指定子を除いたキーワードを返すこと" do
      form = Pages::SearchForm.new(q: "space:space1 space:space2 テストキーワード")
      expect(form.keyword_without_space_filters).to eq("テストキーワード")
    end

    it "space:指定子のみの場合、空文字列を返すこと" do
      form = Pages::SearchForm.new(q: "space:my-space")
      expect(form.keyword_without_space_filters).to eq("")
    end

    it "space:指定子がない場合、元のキーワードを返すこと" do
      form = Pages::SearchForm.new(q: "テストキーワード")
      expect(form.keyword_without_space_filters).to eq("テストキーワード")
    end

    it "キーワードがnilの場合、空文字列を返すこと" do
      form = Pages::SearchForm.new(q: nil)
      expect(form.keyword_without_space_filters).to eq("")
    end
  end

  describe "#has_space_filters?" do
    it "space:指定子がある場合、trueを返すこと" do
      form = Pages::SearchForm.new(q: "space:my-space テストキーワード")
      expect(form.has_space_filters?).to be true
    end

    it "space:指定子がない場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: "テストキーワード")
      expect(form.has_space_filters?).to be false
    end

    it "キーワードがnilの場合、falseを返すこと" do
      form = Pages::SearchForm.new(q: nil)
      expect(form.has_space_filters?).to be false
    end
  end

  describe "#keywords_without_space_filters" do
    it "space:指定子を除いたキーワードを配列で返すこと" do
      form = Pages::SearchForm.new(q: "space:my-space テスト キーワード")
      expect(form.keywords_without_space_filters).to eq(["テスト", "キーワード"])
    end

    it "複数のspace:指定子を除いたキーワードを配列で返すこと" do
      form = Pages::SearchForm.new(q: "space:space1 space:space2 テスト キーワード")
      expect(form.keywords_without_space_filters).to eq(["テスト", "キーワード"])
    end

    it "space:指定子のみの場合、空配列を返すこと" do
      form = Pages::SearchForm.new(q: "space:my-space")
      expect(form.keywords_without_space_filters).to eq([])
    end

    it "space:指定子がない場合、キーワードを配列で返すこと" do
      form = Pages::SearchForm.new(q: "テスト キーワード")
      expect(form.keywords_without_space_filters).to eq(["テスト", "キーワード"])
    end

    it "キーワードがnilの場合、空配列を返すこと" do
      form = Pages::SearchForm.new(q: nil)
      expect(form.keywords_without_space_filters).to eq([])
    end
  end
end
