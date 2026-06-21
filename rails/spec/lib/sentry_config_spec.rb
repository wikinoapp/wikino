# typed: false
# frozen_string_literal: true

RSpec.describe SentryConfig do
  describe ".resolve_traces_sample_rate" do
    it "Float 値はそのまま採用する" do
      expect(described_class.resolve_traces_sample_rate(0.7)).to eq(0.7)
    end

    it "Integer 値は Float に変換して採用する" do
      expect(described_class.resolve_traces_sample_rate(1)).to eq(1.0)
    end

    it "範囲内の数値文字列を採用する" do
      expect(described_class.resolve_traces_sample_rate("0.3")).to eq(0.3)
    end

    it "nil は既定値 0.5 にフォールバックする" do
      expect(described_class.resolve_traces_sample_rate(nil)).to eq(0.5)
    end

    it "空文字列は既定値 0.5 にフォールバックする" do
      expect(described_class.resolve_traces_sample_rate("")).to eq(0.5)
    end

    it "非数値文字列は既定値 0.5 にフォールバックする" do
      expect(described_class.resolve_traces_sample_rate("abc")).to eq(0.5)
    end

    it "範囲外の数値 (1.5) は既定値 0.5 にフォールバックする" do
      expect(described_class.resolve_traces_sample_rate(1.5)).to eq(0.5)
    end

    it "範囲外の数値 (-0.1) は既定値 0.5 にフォールバックする" do
      expect(described_class.resolve_traces_sample_rate(-0.1)).to eq(0.5)
    end

    it "default キーワード引数で既定値を上書きできる" do
      expect(described_class.resolve_traces_sample_rate(nil, default: 0.2)).to eq(0.2)
    end
  end

  describe ".resolve_release" do
    it "コミットハッシュが存在すればそのまま採用する" do
      expect(described_class.resolve_release("1234567")).to eq("1234567")
    end

    it "nil は nil を返す (release を設定せず SDK の自動検出に委ねる)" do
      expect(described_class.resolve_release(nil)).to be_nil
    end

    it "空文字列は nil を返す (release を空文字で上書きしない)" do
      expect(described_class.resolve_release("")).to be_nil
    end

    it "空白のみの文字列は nil を返す" do
      expect(described_class.resolve_release("   ")).to be_nil
    end
  end
end
