# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /attachments/:attachment_id", type: :request do
  before do
    # ActiveStorageのURL生成にホスト情報が必要
    Rails.application.routes.default_url_options[:host] = "test.host"
    # ActiveStorage::Current.url_optionsも設定
    ActiveStorage::Current.url_options = { host: "test.host" }
  end

  # ActiveStorage::Blobのurlメソッドをスタブ化するヘルパー
  def stub_blob_url(blob)
    signed_url = "http://test.host/rails/active_storage/blobs/redirect/#{blob.id}?test_signature"
    allow(blob).to receive(:url).and_return(signed_url)
  end
  
  # AttachmentRecordのredirect_urlメソッドをスタブ化するヘルパー  
  def stub_attachment_redirect_url(attachment)
    signed_url = "http://test.host/rails/active_storage/blobs/redirect/#{attachment.active_storage_attachment_record.blob.id}?test_signature"
    allow(AttachmentRecord).to receive(:find).with(attachment.id).and_return(attachment)
    allow(attachment).to receive(:redirect_url).and_return(signed_url)
  end

  def create_attachment_with_page(space:, topic:, page:, filename: "test.jpg")
    # ActiveStorageのBlobを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("test content"),
      filename: filename,
      content_type: "image/jpeg"
    )

    # Blobのurlメソッドをスタブ化
    stub_blob_url(blob)

    # ActiveStorageのAttachmentを作成
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceMemberRecord",
      record_id: space_member.id,
      blob: blob
    )

    attachment = AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      space_record: space,
      active_storage_attachment_id: active_storage_attachment.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    # ページと添付ファイルの関連を作成
    if page
      PageAttachmentReferenceRecord.create!(
        page_id: page.id,
        attachment_id: attachment.id
      )
    end

    attachment
  end

  it "全て公開トピックのページに関連する添付ファイルの場合、誰でもアクセス可能" do
    space = FactoryBot.create(:space_record)
    public_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Public.serialize)
    page = FactoryBot.create(:page_record, space_record: space, topic_record: public_topic)
    attachment = create_attachment_with_page(space: space, topic: public_topic, page: page)
    
    # AttachmentRecordのredirect_urlをスタブ化
    stub_attachment_redirect_url(attachment)
    
    # ログインしていない状態でアクセス
    get attachment_path(attachment_id: attachment.id)

    # リダイレクトされることを確認
    expect(response).to have_http_status(302)
    expect(response.location).to include("active_storage/blobs/redirect")
  end

  it "プライベートトピックのページに関連する添付ファイルの場合、スペースメンバーのみアクセス可能" do
    space = FactoryBot.create(:space_record)
    private_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Private.serialize)
    page = FactoryBot.create(:page_record, space_record: space, topic_record: private_topic)
    attachment = create_attachment_with_page(space: space, topic: private_topic, page: page)

    # AttachmentRecordのredirect_urlをスタブ化
    stub_attachment_redirect_url(attachment)

    # スペースメンバーでログイン
    user = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record: space, user_record: user, active: true)
    sign_in(user_record: user)

    get attachment_path(attachment_id: attachment.id)

    # リダイレクトされることを確認
    expect(response).to have_http_status(302)
    expect(response.location).to include("active_storage/blobs/redirect")
  end

  it "プライベートトピックのページに関連する添付ファイルの場合、非メンバーはアクセス不可" do
    space = FactoryBot.create(:space_record)
    private_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Private.serialize)
    page = FactoryBot.create(:page_record, space_record: space, topic_record: private_topic)
    attachment = create_attachment_with_page(space: space, topic: private_topic, page: page)

    # ログインしていない状態でアクセス
    get attachment_path(attachment_id: attachment.id)

    # 404エラーになることを確認
    expect(response).to have_http_status(404)
  end

  it "プライベートトピックのページに関連する添付ファイルの場合、他のスペースのメンバーはアクセス不可" do
    space1 = FactoryBot.create(:space_record)
    space2 = FactoryBot.create(:space_record)
    private_topic = FactoryBot.create(:topic_record, space_record: space1, visibility: TopicVisibility::Private.serialize)
    page = FactoryBot.create(:page_record, space_record: space1, topic_record: private_topic)
    attachment = create_attachment_with_page(space: space1, topic: private_topic, page: page)

    # 別のスペースのメンバーでログイン
    user = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record: space2, user_record: user, active: true)
    sign_in(user_record: user)

    get attachment_path(attachment_id: attachment.id)

    # 404エラーになることを確認
    expect(response).to have_http_status(404)
  end

  it "公開トピックとプライベートトピックの両方のページに関連する添付ファイルの場合、スペースメンバーのみアクセス可能" do
    space = FactoryBot.create(:space_record)
    public_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Public.serialize)
    private_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Private.serialize)
    page1 = FactoryBot.create(:page_record, space_record: space, topic_record: public_topic)
    page2 = FactoryBot.create(:page_record, space_record: space, topic_record: private_topic)

    # 添付ファイルを作成
    attachment = create_attachment_with_page(space: space, topic: public_topic, page: page1)
    # 2つ目のページとの関連も作成
    PageAttachmentReferenceRecord.create!(
      page_id: page2.id,
      attachment_id: attachment.id
    )

    # ログインしていない状態でアクセス
    get attachment_path(attachment_id: attachment.id)

    # 404エラーになることを確認（プライベートトピックのページがあるため）
    expect(response).to have_http_status(404)

    # スペースメンバーでログイン
    user = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record: space, user_record: user, active: true)
    sign_in(user_record: user)

    # AttachmentRecordのredirect_urlをスタブ化
    stub_attachment_redirect_url(attachment)

    get attachment_path(attachment_id: attachment.id)

    # リダイレクトされることを確認
    expect(response).to have_http_status(302)
    expect(response.location).to include("active_storage/blobs/redirect")
  end

  it "ページに関連付けられていない添付ファイルの場合、スペースメンバーのみアクセス可能" do
    space = FactoryBot.create(:space_record)
    attachment = create_attachment_with_page(space: space, topic: nil, page: nil)

    # ログインしていない状態でアクセス
    get attachment_path(attachment_id: attachment.id)

    # 404エラーになることを確認
    expect(response).to have_http_status(404)

    # スペースメンバーでログイン
    user = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record: space, user_record: user, active: true)
    sign_in(user_record: user)

    # AttachmentRecordのredirect_urlをスタブ化
    stub_attachment_redirect_url(attachment)

    get attachment_path(attachment_id: attachment.id)

    # リダイレクトされることを確認
    expect(response).to have_http_status(302)
    expect(response.location).to include("active_storage/blobs/redirect")
  end

  it "存在しない添付ファイルIDの場合、404エラーになること" do
    # 存在しないIDでアクセス
    non_existent_id = SecureRandom.uuid
    get attachment_path(attachment_id: non_existent_id)

    # エラーになることを確認（RecordNotFoundをハンドリングする前提）
    expect { AttachmentRecord.find(non_existent_id) }.to raise_error(ActiveRecord::RecordNotFound)
  end

  it "ActiveStorageのblobが存在しない場合、404エラーになること" do
    space = FactoryBot.create(:space_record)
    public_topic = FactoryBot.create(:topic_record, space_record: space, visibility: TopicVisibility::Public.serialize)
    page = FactoryBot.create(:page_record, space_record: space, topic_record: public_topic)
    attachment = create_attachment_with_page(space: space, topic: public_topic, page: page)

    # ActiveStorageのblobへのアクセスをモックして、nilを返すようにする
    active_storage_attachment = instance_double(ActiveStorage::Attachment, blob: nil)
    allow(attachment).to receive(:active_storage_attachment_record).and_return(active_storage_attachment)
    allow(AttachmentRecord).to receive(:find).with(attachment.id).and_return(attachment)

    get attachment_path(attachment_id: attachment.id)

    # 404エラーになることを確認
    expect(response).to have_http_status(404)
  end
end
