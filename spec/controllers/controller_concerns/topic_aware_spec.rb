# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe ControllerConcerns::TopicAware do
  # テスト用のコントローラークラス
  class TestController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::TopicAware
  end

  it "ヘルパーメソッドが使用できること" do
    controller = TestController.new

    # TopicAwareのメソッドが定義されていること
    expect(controller).to respond_to(:current_topic_member_record)
    expect(controller).to respond_to(:current_topic_member_record!)
    expect(controller).to respond_to(:current_topic_record)
    expect(controller).to respond_to(:current_topic_record!)
    expect(controller).to respond_to(:topic_policy_for)

    # SpaceAwareのメソッドも使用できること
    expect(controller).to respond_to(:current_space_member_record)
    expect(controller).to respond_to(:space_policy_for)
  end

  it "current_topic_member_recordが正しく動作すること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_member_record = FactoryBot.create(:topic_member_record,
      space_member_record:,
      topic_record:)

    controller = TestController.new
    allow(controller).to receive(:current_user_record).and_return(user_record)

    result = controller.current_topic_member_record(topic_record:)
    expect(result).to eq(topic_member_record)
  end

  it "topic_policy_forがSpace Ownerに対して正しいPolicyを返すこと" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:space_member_record,
      user_record:,
      space_record:,
      role: SpaceMemberRole::Owner.serialize)

    controller = TestController.new
    allow(controller).to receive(:current_user_record).and_return(user_record)

    policy = controller.topic_policy_for(topic_record:)
    expect(policy).to be_a(SpaceOwnerPolicy)
  end

  it "topic_policy_forがTopic Adminに対して正しいPolicyを返すこと" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record,
      user_record:,
      space_record:,
      role: SpaceMemberRole::Member.serialize)
    FactoryBot.create(:topic_member_record,
      space_member_record:,
      topic_record:,
      role: TopicMemberRole::Admin.serialize)

    controller = TestController.new
    allow(controller).to receive(:current_user_record).and_return(user_record)

    policy = controller.topic_policy_for(topic_record:)
    expect(policy).to be_a(TopicAdminPolicy)
  end

  it "current_topic_recordがパラメータからTopicレコードを取得すること" do
    space_record = FactoryBot.create(:space_record, identifier: "test-space")
    topic_record = FactoryBot.create(:topic_record, space_record:, number: 1)

    controller = TestController.new
    allow(controller).to receive(:params).and_return({
      space_identifier: "test-space",
      topic_number: "1"  # Topicはnumberで識別される
    })

    result = controller.current_topic_record
    expect(result).to eq(topic_record)
  end
end
