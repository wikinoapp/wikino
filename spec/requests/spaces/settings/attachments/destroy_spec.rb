# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "DELETE /s/:space_identifier/settings/attachments/:attachment_id", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトされること" do
    space_record = FactoryBot.create(:space_record)
    attachment_record = FactoryBot.create(:attachment_record, space_record:)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースメンバーでない場合、404エラーになること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    attachment_record = FactoryBot.create(:attachment_record, space_record:)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(404)
  end

  it "添付ファイルの削除権限がない場合、404エラーになること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record:, user_record:, role: SpaceMemberRole::Member.serialize)
    
    # 他のメンバーがアップロードしたファイル
    other_member = FactoryBot.create(:space_member_record, space_record:)
    attachment_record = FactoryBot.create(:attachment_record, space_record:, attached_space_member_record: other_member)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(404)
  end

  it "添付ファイルを削除できること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record:, user_record:, role: SpaceMemberRole::Admin.serialize)
    attachment_record = FactoryBot.create(:attachment_record, space_record:, attached_space_member_record: space_member_record)

    # 削除サービスがモック化されるように
    delete_service = instance_double(Attachments::DeleteService)
    allow(Attachments::DeleteService).to receive(:new).and_return(delete_service)
    allow(delete_service).to receive(:call).with(attachment_record_id: attachment_record.id)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}/settings/attachments")
    expect(flash[:notice]).to eq(I18n.t("messages.attachments.deleted_successfully"))
    expect(delete_service).to have_received(:call).with(attachment_record_id: attachment_record.id)
  end

  it "自分がアップロードした添付ファイルは一般メンバーでも削除できること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record:, user_record:, role: SpaceMemberRole::Member.serialize)
    
    # 自分がアップロードしたファイル
    attachment_record = FactoryBot.create(:attachment_record, space_record:, attached_space_member_record: space_member_record)

    # 削除サービスがモック化されるように
    delete_service = instance_double(Attachments::DeleteService)
    allow(Attachments::DeleteService).to receive(:new).and_return(delete_service)
    allow(delete_service).to receive(:call).with(attachment_record_id: attachment_record.id)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}/settings/attachments")
    expect(delete_service).to have_received(:call).with(attachment_record_id: attachment_record.id)
  end
end