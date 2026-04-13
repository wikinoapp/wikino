# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe MemberPolicy do
  describe "#can_show_topic?" do
    it "公開トピックはスコープなしでも閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)

      policy = MemberPolicy.new(space_scopes: [], topic_scopes: [])
      expect(policy.can_show_topic?(topic_record:)).to be(true)
    end

    it "非公開トピックはtopic:readで閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)

      policy = MemberPolicy.new(
        space_scopes: [],
        topic_scopes: [Scope::TOPIC_READ]
      )
      expect(policy.can_show_topic?(topic_record:)).to be(true)
    end

    it "非公開トピックはtopic:readなしで閲覧不可であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)

      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_show_topic?(topic_record:)).to be(false)
    end

    it "space:adminは非公開トピックを閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)

      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_show_topic?(topic_record:)).to be(true)
    end
  end

  describe "#can_update_topic?" do
    it "topic:writeでトピックを更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_topic?).to be(true)
    end

    it "topic:writeなしでトピックを更新不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_READ],
        topic_scopes: []
      )
      expect(policy.can_update_topic?).to be(false)
    end

    it "space:adminでトピックを更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_update_topic?).to be(true)
    end
  end

  describe "#can_delete_topic?" do
    it "topic:deleteでトピックを削除可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_DELETE],
        topic_scopes: []
      )
      expect(policy.can_delete_topic?).to be(true)
    end

    it "topic:deleteなしでトピックを削除不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_WRITE],
        topic_scopes: []
      )
      expect(policy.can_delete_topic?).to be(false)
    end

    it "space:adminでトピックを削除可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_delete_topic?).to be(true)
    end
  end

  describe "#can_create_page?" do
    it "page:writeでページを作成可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_create_page?).to be(true)
    end

    it "page:writeなしでページを作成不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_READ],
        topic_scopes: []
      )
      expect(policy.can_create_page?).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "page:writeでページを更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_page?).to be(true)
    end

    it "page:writeなしでページを更新不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_READ],
        topic_scopes: []
      )
      expect(policy.can_update_page?).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "page:trashでページをゴミ箱に移動可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_TRASH],
        topic_scopes: []
      )
      expect(policy.can_trash_page?).to be(true)
    end

    it "page:trashなしでページをゴミ箱に移動不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_trash_page?).to be(false)
    end

    it "space:adminでページをゴミ箱に移動可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_trash_page?).to be(true)
    end
  end

  describe "#can_show_draft_page?" do
    it "所有者はdraft_page:readで閲覧可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::DRAFT_PAGE_READ],
        topic_scopes: []
      )
      expect(policy.can_show_draft_page?(is_owner: true)).to be(true)
    end

    it "非所有者はdraft_page:readだけでは閲覧不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::DRAFT_PAGE_READ],
        topic_scopes: []
      )
      expect(policy.can_show_draft_page?(is_owner: false)).to be(false)
    end

    it "非所有者でもspace:adminなら閲覧可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_show_draft_page?(is_owner: false)).to be(true)
    end

    it "draft_page:readなしでは所有者でも閲覧不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_READ],
        topic_scopes: []
      )
      expect(policy.can_show_draft_page?(is_owner: true)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "所有者はdraft_page:writeで更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::DRAFT_PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_draft_page?(is_owner: true)).to be(true)
    end

    it "非所有者はdraft_page:writeだけでは更新不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::DRAFT_PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_draft_page?(is_owner: false)).to be(false)
    end

    it "非所有者でもspace:adminなら更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_update_draft_page?(is_owner: false)).to be(true)
    end

    it "draft_page:writeなしでは所有者でも更新不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_draft_page?(is_owner: true)).to be(false)
    end
  end

  describe "#can_update_space?" do
    it "space:writeでスペースを更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_update_space?).to be(true)
    end

    it "space:writeなしでスペースを更新不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_READ],
        topic_scopes: []
      )
      expect(policy.can_update_space?).to be(false)
    end

    it "space:adminでスペースを更新可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_update_space?).to be(true)
    end
  end

  describe "#can_export_space?" do
    it "space:writeでスペースをエクスポート可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_export_space?).to be(true)
    end

    it "space:writeなしでスペースをエクスポート不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_READ],
        topic_scopes: []
      )
      expect(policy.can_export_space?).to be(false)
    end

    it "space:adminでスペースをエクスポート可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_export_space?).to be(true)
    end
  end

  describe "#can_create_topic?" do
    it "topic:writeでトピックを作成可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_WRITE],
        topic_scopes: []
      )
      expect(policy.can_create_topic?).to be(true)
    end

    it "topic:writeなしでトピックを作成不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::TOPIC_READ],
        topic_scopes: []
      )
      expect(policy.can_create_topic?).to be(false)
    end

    it "space:adminでトピックを作成可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_create_topic?).to be(true)
    end
  end

  describe "#can_show_trash?" do
    it "page:trashでゴミ箱を閲覧可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_TRASH],
        topic_scopes: []
      )
      expect(policy.can_show_trash?).to be(true)
    end

    it "page:trashなしでゴミ箱を閲覧不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_show_trash?).to be(false)
    end

    it "space:adminでゴミ箱を閲覧可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_show_trash?).to be(true)
    end
  end

  describe "#can_create_bulk_restore_pages?" do
    it "page:restoreで一括復元可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_RESTORE],
        topic_scopes: []
      )
      expect(policy.can_create_bulk_restore_pages?).to be(true)
    end

    it "page:restoreなしで一括復元不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::PAGE_WRITE],
        topic_scopes: []
      )
      expect(policy.can_create_bulk_restore_pages?).to be(false)
    end

    it "space:adminで一括復元可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_create_bulk_restore_pages?).to be(true)
    end
  end

  describe "#can_upload_attachment?" do
    it "attachment:writeでファイルアップロード可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::ATTACHMENT_WRITE],
        topic_scopes: []
      )
      expect(policy.can_upload_attachment?).to be(true)
    end

    it "attachment:writeなしでファイルアップロード不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::ATTACHMENT_READ],
        topic_scopes: []
      )
      expect(policy.can_upload_attachment?).to be(false)
    end

    it "space:adminでファイルアップロード可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_upload_attachment?).to be(true)
    end
  end

  describe "#can_view_attachment?" do
    it "attachment:readで添付ファイルを閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)

      policy = MemberPolicy.new(
        space_scopes: [Scope::ATTACHMENT_READ],
        topic_scopes: []
      )
      expect(policy.can_view_attachment?(attachment_record:)).to be(true)
    end

    it "attachment:readなしでも公開ページから参照されていれば閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)
      FactoryBot.create(:page_attachment_reference_record, page_record:, attachment_record:)

      policy = MemberPolicy.new(
        space_scopes: [],
        topic_scopes: []
      )
      expect(policy.can_view_attachment?(attachment_record:)).to be(true)
    end

    it "attachment:readなしかつ公開ページからの参照もなければ閲覧不可であること" do
      space_record = FactoryBot.create(:space_record)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)

      policy = MemberPolicy.new(
        space_scopes: [],
        topic_scopes: []
      )
      expect(policy.can_view_attachment?(attachment_record:)).to be(false)
    end

    it "space:adminで添付ファイルを閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)
      attachment_record = FactoryBot.create(:attachment_record, space_record:)

      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_view_attachment?(attachment_record:)).to be(true)
    end
  end

  describe "#can_delete_attachment?" do
    it "attachment:deleteでファイル削除可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::ATTACHMENT_DELETE],
        topic_scopes: []
      )
      expect(policy.can_delete_attachment?).to be(true)
    end

    it "attachment:deleteなしでファイル削除不可であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::ATTACHMENT_WRITE],
        topic_scopes: []
      )
      expect(policy.can_delete_attachment?).to be(false)
    end

    it "space:adminでファイル削除可能であること" do
      policy = MemberPolicy.new(
        space_scopes: [Scope::SPACE_ADMIN],
        topic_scopes: []
      )
      expect(policy.can_delete_attachment?).to be(true)
    end
  end
end
