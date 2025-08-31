# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe QueryOptimizer do
  # テスト用のポリシークラス
  class OptimizedTestPolicy < ApplicationPolicy
    include QueryOptimizer

    def initialize(user_record:)
      super
    end
  end

  it "SpaceMemberRecordの関連をプリロードすること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    policy = OptimizedTestPolicy.new(user_record:)

    # プリロード前
    original_record = SpaceMemberRecord.find(space_member_record.id)
    expect(original_record.association(:space_record).loaded?).to eq(false)
    expect(original_record.association(:user_record).loaded?).to eq(false)

    # プリロード実行
    optimized_record = policy.preload_space_member_associations(original_record)

    # プリロード後
    expect(optimized_record).not_to be_nil
    expect(optimized_record.association(:space_record).loaded?).to eq(true)
    expect(optimized_record.association(:user_record).loaded?).to eq(true)
    expect(optimized_record.association(:topic_member_records).loaded?).to eq(true)
  end

  it "TopicMemberRecordの関連をプリロードすること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    topic_member_record = FactoryBot.create(:topic_member_record, topic_record:, space_member_record:)
    policy = OptimizedTestPolicy.new(user_record:)

    # プリロード前
    original_record = TopicMemberRecord.find(topic_member_record.id)
    expect(original_record.association(:topic_record).loaded?).to eq(false)
    expect(original_record.association(:space_member_record).loaded?).to eq(false)

    # プリロード実行
    optimized_record = policy.preload_topic_member_associations(original_record)

    # プリロード後
    expect(optimized_record).not_to be_nil
    expect(optimized_record.association(:topic_record).loaded?).to eq(true)
    expect(optimized_record.association(:space_member_record).loaded?).to eq(true)
  end

  it "PageRecordの関連をプリロードすること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    page_record = FactoryBot.create(:page_record, topic_record:)
    policy = OptimizedTestPolicy.new(user_record:)

    # プリロード前
    original_record = PageRecord.find(page_record.id)
    expect(original_record.association(:topic_record).loaded?).to eq(false)

    # プリロード実行
    optimized_record = policy.preload_page_associations(original_record)

    # プリロード後
    expect(optimized_record).not_to be_nil
    expect(optimized_record.association(:topic_record).loaded?).to eq(true)
    # ネストした関連もプリロードされる
    expect(optimized_record.topic_record.association(:space_record).loaded?).to eq(true)
  end

  it "複数のPageRecordをバッチでプリロードすること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    page_records = 3.times.map do
      FactoryBot.create(:page_record, topic_record:)
    end
    policy = OptimizedTestPolicy.new(user_record:)

    # プリロード実行
    optimized_records = policy.preload_pages_for_permission_check(page_records)

    # 全てのレコードがプリロードされていること
    expect(optimized_records.size).to eq(3)
    optimized_records.each do |record|
      expect(record.association(:space_record).loaded?).to eq(true)
      expect(record.association(:topic_record).loaded?).to eq(true)
      expect(record.topic_record.association(:space_record).loaded?).to eq(true)
      expect(record.topic_record.association(:member_records).loaded?).to eq(true)
    end
  end

  it "nilの場合はnilを返すこと" do
    user_record = FactoryBot.create(:user_record)
    policy = OptimizedTestPolicy.new(user_record:)

    expect(policy.preload_space_member_associations(nil)).to be_nil
    expect(policy.preload_topic_member_associations(nil)).to be_nil
  end
end

