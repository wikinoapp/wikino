# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "config/initializers/active_storage_analyze_job.rb" do # rubocop:disable RSpec/DescribeClass
  describe "ActiveStorage::AnalyzeJob" do
    it "ActiveStorage::FileNotFoundError 用の rescue ハンドラを登録していること" do
      handler = ActiveStorage::AnalyzeJob.rescue_handlers.find do |class_name, _|
        class_name == "ActiveStorage::FileNotFoundError"
      end

      expect(handler).to be_present
    end

    it "FileNotFoundError を discard し、再 raise しないこと" do
      job = ActiveStorage::AnalyzeJob.new

      # rescue_with_handler はマッチするハンドラがあれば実行結果 (truthy) を返し、
      # 無ければ nil を返す。discard されること (= 再 raise されないこと) を検証する。
      # [Ja] rescue_with_handler は該当ハンドラがあれば実行結果 (truthy) を、無ければ
      # nil を返す。例外が discard される (再 raise されない) ことを確認する。
      handled = nil
      expect {
        handled = job.rescue_with_handler(ActiveStorage::FileNotFoundError.new)
      }.not_to raise_error

      expect(handled).to be_truthy
    end
  end
end
