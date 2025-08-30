# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe ControllerConcerns::SpaceAware do
  # テスト用のコントローラークラス
  class TestController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::SpaceAware
  end

  it "ヘルパーメソッドが使用できること" do
    controller = TestController.new

    # current_space_member_recordメソッドが定義されていること
    expect(controller).to respond_to(:current_space_member_record)
    expect(controller).to respond_to(:current_space_member_record!)
    expect(controller).to respond_to(:space_policy_for)
    expect(controller).to respond_to(:current_space_record)
    expect(controller).to respond_to(:current_space_record!)
  end

  it "current_space_member_recordが正しく動作すること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

    controller = TestController.new
    allow(controller).to receive(:current_user_record).and_return(user_record)

    result = controller.current_space_member_record(space_record:)
    expect(result).to eq(space_member_record)
  end

  it "space_policy_forが正しいPolicyを返すこと" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, 
      user_record:, 
      space_record:,
      role: SpaceMemberRole::Owner.serialize
    )

    controller = TestController.new
    allow(controller).to receive(:current_user_record).and_return(user_record)

    policy = controller.space_policy_for(space_record:)
    expect(policy).to be_a(SpaceOwnerPolicy)
  end

  it "current_space_recordがパラメータからSpaceレコードを取得すること" do
    space_record = FactoryBot.create(:space_record, identifier: "test-space")

    controller = TestController.new
    allow(controller).to receive(:params).and_return({ space_identifier: "test-space" })

    result = controller.current_space_record
    expect(result).to eq(space_record)
  end
end