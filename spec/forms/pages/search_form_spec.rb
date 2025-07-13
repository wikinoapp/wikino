# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe Pages::SearchForm, type: :model do
  around do |example|
    I18n.with_locale(:ja) do
      example.run
    end
  end
  describe "バリデーション" do
    it "有効なキーワードの場合、バリデーションが通ること" do
      form = described_class.new(q: "テストキーワード")
      expect(form).to be_valid
    end

    it "空文字列の場合、バリデーションが通ること" do
      form = described_class.new(q: "")
      expect(form).to be_valid
    end

    it "nilの場合、バリデーションが通ること" do
      form = described_class.new(q: nil)
      expect(form).to be_valid
    end

    it "1文字の場合、バリデーションエラーになること" do
      form = described_class.new(q: "a")
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("は2文字以上で入力してください")
    end

    it "101文字以上の場合、バリデーションエラーになること" do
      form = described_class.new(q: "a" * 101)
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("は100文字以内で入力してください")
    end

    it "不正な文字(<や>)が含まれている場合、バリデーションエラーになること" do
      form = described_class.new(q: "test<script>")
      expect(form).not_to be_valid
      expect(form.errors[:q]).to include("に不正な文字が含まれています")
    end
  end

  describe "#q_present?" do
    it "キーワードが存在する場合、trueを返すこと" do
      form = described_class.new(q: "テスト")
      expect(form.q_present?).to be true
    end

    it "キーワードが空文字列の場合、falseを返すこと" do
      form = described_class.new(q: "")
      expect(form.q_present?).to be false
    end

    it "キーワードがnilの場合、falseを返すこと" do
      form = described_class.new(q: nil)
      expect(form.q_present?).to be false
    end
  end

  describe "#searchable?" do
    it "有効でキーワードが存在する場合、trueを返すこと" do
      form = described_class.new(q: "テスト")
      expect(form.searchable?).to be true
    end

    it "無効な場合、falseを返すこと" do
      form = described_class.new(q: "a")
      expect(form.searchable?).to be false
    end

    it "キーワードが存在しない場合、falseを返すこと" do
      form = described_class.new(q: "")
      expect(form.searchable?).to be false
    end
  end
end