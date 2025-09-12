# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicRepository do
  describe "#find_topics_by_space" do
    it "スペースに参加しているトピックをlast_page_modified_atの降順で返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)

      # 3つのトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic 1")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic 2")
      topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Topic 3")

      # トピックメンバーを作成（last_page_modified_atを設定）
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize,
        last_page_modified_at: 3.days.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record:,
        role: TopicMemberRole::Member.serialize,
        last_page_modified_at: 1.day.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record3,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize,
        last_page_modified_at: 2.days.ago)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record:)

      # 検証
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("Topic 2") # 1日前（最新）
      expect(topics[1].name).to eq("Topic 3") # 2日前
      expect(topics[2].name).to eq("Topic 1") # 3日前（最古）
    end

    it "space_member_recordがnilの場合、空配列を返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_record:, topic_record:, space_member_record:)

      # メソッド実行（space_member_record: nil）
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record: nil)

      # 検証
      expect(topics).to be_empty
    end

    it "ユーザーが参加していないトピックは返さないこと" do
      # テスト用のデータ作成
      user_record1 = FactoryBot.create(:user_record)
      user_record2 = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record1 = FactoryBot.create(:space_member_record, :member, user_record: user_record1, space_record:)
      space_member_record2 = FactoryBot.create(:space_member_record, :member, user_record: user_record2, space_record:)

      # 3つのトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic 1")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic 2")
      topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Topic 3")

      # user_record1は topic1とtopic2に参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record: space_member_record1)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record: space_member_record1)

      # user_record2はtopic3のみに参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record3,
        space_member_record: space_member_record2)

      # user_record1でメソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record: space_member_record1)

      # 検証: user_record1が参加している2つのトピックのみが返される
      expect(topics.size).to eq(2)
      expect(topics.map(&:name)).to contain_exactly("Topic 1", "Topic 2")
    end

    it "ユーザーがスペースメンバーでない場合、空配列を返すこと" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:topic_record, space_record:)

      # space_member_recordはnil（スペースメンバーではない）

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record: nil)

      # 検証
      expect(topics).to be_empty
    end

    it "トピック管理者の場合、can_updateがtrueになること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record:)

      # 検証
      expect(topics.size).to eq(1)
      topic = topics.first
      expect(topic.can_update).to be(true)
      expect(topic.can_create_page).to be(true)
    end

    it "トピック一般メンバーの場合、can_updateがfalse、can_create_pageがtrueになること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record:)

      # 検証
      expect(topics.size).to eq(1)
      topic = topics.first
      expect(topic.can_update).to be(false)
      expect(topic.can_create_page).to be(true)
    end

    it "last_page_modified_atがnilのトピックは最後に表示されること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)

      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Topic with nil")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Topic with date")

      # 1つ目のトピックメンバーはlast_page_modified_atがnil
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        last_page_modified_at: nil)
      # 2つ目のトピックメンバーはlast_page_modified_atが設定されている
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record:,
        last_page_modified_at: 1.day.ago)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_topics_by_space(space_record:, space_member_record:)

      # 検証
      expect(topics.size).to eq(2)
      expect(topics[0].name).to eq("Topic with date")
      expect(topics[1].name).to eq("Topic with nil")
    end
  end

  describe "#find_public_topics_by_space" do
    it "公開トピックのみを返すこと" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)

      # 公開トピックと非公開トピックを作成
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Public Topic 1",
        visibility: TopicVisibility::Public.serialize)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Private Topic",
        visibility: TopicVisibility::Private.serialize)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Public Topic 2",
        visibility: TopicVisibility::Public.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_public_topics_by_space(space_record:)

      # 検証：公開トピックのみが返される
      expect(topics.size).to eq(2)
      expect(topics.map(&:name)).to contain_exactly("Public Topic 1", "Public Topic 2")
    end

    it "作成日時の降順でソートされること" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)

      # 異なる作成日時で公開トピックを作成
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Old Topic",
        visibility: TopicVisibility::Public.serialize,
        created_at: 3.days.ago)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "New Topic",
        visibility: TopicVisibility::Public.serialize,
        created_at: 1.day.ago)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Middle Topic",
        visibility: TopicVisibility::Public.serialize,
        created_at: 2.days.ago)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_public_topics_by_space(space_record:)

      # 検証：新しい順にソートされている
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("New Topic")
      expect(topics[1].name).to eq("Middle Topic")
      expect(topics[2].name).to eq("Old Topic")
    end

    it "権限フラグがすべてfalseになること" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_public_topics_by_space(space_record:)

      # 検証：権限フラグがすべてfalse
      expect(topics.size).to eq(1)
      topic = topics.first
      expect(topic.can_update).to be(false)
      expect(topic.can_create_page).to be(false)
    end

    it "公開トピックがない場合、空配列を返すこと" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)
      # 非公開トピックのみ作成
      FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_public_topics_by_space(space_record:)

      # 検証
      expect(topics).to be_empty
    end

    it "作成日時が同じ場合、トピック番号の降順でソートされること" do
      # テスト用のデータ作成
      space_record = FactoryBot.create(:space_record)
      same_time = 1.day.ago

      # 同じ作成日時で異なるnumberのトピックを作成
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Topic Number 1",
        number: 1,
        visibility: TopicVisibility::Public.serialize,
        created_at: same_time)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Topic Number 3",
        number: 3,
        visibility: TopicVisibility::Public.serialize,
        created_at: same_time)
      FactoryBot.create(:topic_record,
        space_record:,
        name: "Topic Number 2",
        number: 2,
        visibility: TopicVisibility::Public.serialize,
        created_at: same_time)

      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_public_topics_by_space(space_record:)

      # 検証：numberの降順でソート
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("Topic Number 3")
      expect(topics[1].name).to eq("Topic Number 2")
      expect(topics[2].name).to eq("Topic Number 1")
    end
  end

  describe "#find_joined_topics" do
    it "ユーザーが参加している全スペースの全トピックを返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      
      # 2つのスペースを作成
      space_record1 = FactoryBot.create(:space_record, name: "Space 1")
      space_record2 = FactoryBot.create(:space_record, name: "Space 2")
      
      # ユーザーを両方のスペースのメンバーにする
      space_member_record1 = FactoryBot.create(:space_member_record, :member, user_record:, space_record: space_record1)
      space_member_record2 = FactoryBot.create(:space_member_record, :member, user_record:, space_record: space_record2)
      
      # 各スペースにトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record: space_record1, name: "Topic 1 in Space 1")
      topic_record2 = FactoryBot.create(:topic_record, space_record: space_record1, name: "Topic 2 in Space 1")
      topic_record3 = FactoryBot.create(:topic_record, space_record: space_record2, name: "Topic 1 in Space 2")
      
      # ユーザーを各トピックのメンバーにする
      FactoryBot.create(:topic_member_record,
        space_record: space_record1,
        topic_record: topic_record1,
        space_member_record: space_member_record1,
        last_page_modified_at: 3.days.ago)
      FactoryBot.create(:topic_member_record,
        space_record: space_record1,
        topic_record: topic_record2,
        space_member_record: space_member_record1,
        last_page_modified_at: 1.day.ago)
      FactoryBot.create(:topic_member_record,
        space_record: space_record2,
        topic_record: topic_record3,
        space_member_record: space_member_record2,
        last_page_modified_at: 2.days.ago)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証
      expect(topics.size).to eq(3)
      expect(topics.map(&:name)).to contain_exactly("Topic 1 in Space 1", "Topic 2 in Space 1", "Topic 1 in Space 2")
    end

    it "last_page_modified_atの降順でソートされること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      
      # 3つのトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record:, name: "Oldest Topic")
      topic_record2 = FactoryBot.create(:topic_record, space_record:, name: "Newest Topic")
      topic_record3 = FactoryBot.create(:topic_record, space_record:, name: "Middle Topic")
      
      # 異なるlast_page_modified_atで参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        last_page_modified_at: 3.days.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record2,
        space_member_record:,
        last_page_modified_at: 1.day.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record3,
        space_member_record:,
        last_page_modified_at: 2.days.ago)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証：新しい順にソート
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("Newest Topic")
      expect(topics[1].name).to eq("Middle Topic")
      expect(topics[2].name).to eq("Oldest Topic")
    end

    it "非アクティブなスペースメンバーのトピックは含まれないこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record1 = FactoryBot.create(:space_record)
      space_record2 = FactoryBot.create(:space_record)
      
      # アクティブなスペースメンバー
      active_space_member = FactoryBot.create(:space_member_record, :member, 
        user_record:, 
        space_record: space_record1,
        active: true)
      
      # 非アクティブなスペースメンバー
      inactive_space_member = FactoryBot.create(:space_member_record, :member,
        user_record:,
        space_record: space_record2,
        active: false)
      
      # 各スペースにトピックを作成
      topic_record1 = FactoryBot.create(:topic_record, space_record: space_record1, name: "Active Space Topic")
      topic_record2 = FactoryBot.create(:topic_record, space_record: space_record2, name: "Inactive Space Topic")
      
      # トピックメンバーを作成
      FactoryBot.create(:topic_member_record,
        space_record: space_record1,
        topic_record: topic_record1,
        space_member_record: active_space_member)
      FactoryBot.create(:topic_member_record,
        space_record: space_record2,
        topic_record: topic_record2,
        space_member_record: inactive_space_member)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証：アクティブなスペースのトピックのみ
      expect(topics.size).to eq(1)
      expect(topics[0].name).to eq("Active Space Topic")
    end

    it "削除されたトピックは含まれないこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      
      # アクティブなトピックと削除されたトピックを作成
      active_topic = FactoryBot.create(:topic_record, space_record:, name: "Active Topic")
      deleted_topic = FactoryBot.create(:topic_record, space_record:, name: "Deleted Topic")
      deleted_topic.discard!
      
      # 両方のトピックにメンバーとして参加
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: active_topic,
        space_member_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: deleted_topic,
        space_member_record:)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証：アクティブなトピックのみ
      expect(topics.size).to eq(1)
      expect(topics[0].name).to eq("Active Topic")
    end

    it "正しい権限情報が設定されること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      
      # 管理者として参加するトピック
      admin_topic = FactoryBot.create(:topic_record, space_record:, name: "Admin Topic")
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: admin_topic,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)
      
      # メンバーとして参加するトピック
      member_topic = FactoryBot.create(:topic_record, space_record:, name: "Member Topic")
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: member_topic,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証
      expect(topics.size).to eq(2)
      
      admin_topic_model = topics.find { |t| t.name == "Admin Topic" }
      expect(admin_topic_model.can_update).to be(true)
      expect(admin_topic_model.can_create_page).to be(true)
      
      member_topic_model = topics.find { |t| t.name == "Member Topic" }
      expect(member_topic_model.can_update).to be(false)
      expect(member_topic_model.can_create_page).to be(true)
    end

    it "last_page_modified_atがnilの場合は最後にソートされること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      
      # トピックを作成
      topic_with_date = FactoryBot.create(:topic_record, space_record:, name: "Topic with date", number: 1)
      topic_with_nil = FactoryBot.create(:topic_record, space_record:, name: "Topic with nil", number: 2)
      
      # トピックメンバーを作成（一つはlast_page_modified_atがnil）
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_with_date,
        space_member_record:,
        last_page_modified_at: 1.day.ago)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_with_nil,
        space_member_record:,
        last_page_modified_at: nil)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証
      expect(topics.size).to eq(2)
      expect(topics[0].name).to eq("Topic with date")
      expect(topics[1].name).to eq("Topic with nil")
    end

    it "同じlast_page_modified_atの場合、トピック番号の降順でソートされること" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
      same_time = 1.day.ago
      
      # 同じlast_page_modified_atで異なるnumberのトピックを作成
      topic1 = FactoryBot.create(:topic_record, space_record:, name: "Topic Number 1", number: 1)
      topic3 = FactoryBot.create(:topic_record, space_record:, name: "Topic Number 3", number: 3)
      topic2 = FactoryBot.create(:topic_record, space_record:, name: "Topic Number 2", number: 2)
      
      # トピックメンバーを作成（同じlast_page_modified_at）
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic1,
        space_member_record:,
        last_page_modified_at: same_time)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic3,
        space_member_record:,
        last_page_modified_at: same_time)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic2,
        space_member_record:,
        last_page_modified_at: same_time)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証：numberの降順でソート
      expect(topics.size).to eq(3)
      expect(topics[0].name).to eq("Topic Number 3")
      expect(topics[1].name).to eq("Topic Number 2")
      expect(topics[2].name).to eq("Topic Number 1")
    end

    it "参加しているトピックがない場合、空配列を返すこと" do
      # テスト用のデータ作成
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      
      # 他のユーザーだけがメンバーのトピックを作成
      other_space_member = FactoryBot.create(:space_member_record, :member, user_record: other_user_record, space_record:)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record: other_space_member)
      
      # メソッド実行
      repository = TopicRepository.new
      topics = repository.find_joined_topics(user_record:)
      
      # 検証
      expect(topics).to be_empty
    end
  end
end
