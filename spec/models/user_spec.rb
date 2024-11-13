# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_topics" do
    context "トピックが存在するとき" do
      let!(:space) { create(:space) }
      let!(:user) { create(:user, space:) }
      let!(:topic_a) { create(:topic, :public, space:, name: "トピックA") }
      let!(:topic_b) { create(:topic, :private, space:, name: "トピックB") }
      let!(:topic_c) { create(:topic, :private, space:, name: "トピックC") }

      before do
        create(:topic, :private, space:, name: "トピックD")

        create(:topic_membership, :admin, space:, topic: topic_b, member: user)
        create(:topic_membership, :member, space:, topic: topic_c, member: user)
      end

      it "閲覧可能なトピックを返すこと" do
        expect(user.viewable_topics).to contain_exactly(topic_a, topic_b, topic_c)
      end
    end
  end

  describe "#last_page_modified_topics" do
    context "トピックが存在するとき" do
      let!(:space) { create(:space) }
      let!(:user) { create(:user, space:) }
      let!(:topic_a) { create(:topic, space:, name: "トピックA") }
      let!(:topic_b) { create(:topic, space:, name: "トピックB") }
      let!(:topic_c) { create(:topic, space:, name: "トピックC") }
      let!(:topic_d) { create(:topic, space:, name: "トピックD") }

      before do
        create(:topic, space:, name: "トピックE")

        create(:topic_membership, space:, topic: topic_a, member: user, joined_at: Time.zone.parse("2024-08-18 0:00:00"), last_page_modified_at: nil)
        create(:topic_membership, space:, topic: topic_b, member: user, joined_at: Time.zone.parse("2024-08-18 1:00:00"), last_page_modified_at: Time.zone.parse("2024-08-19 0:00:00"))
        create(:topic_membership, space:, topic: topic_c, member: user, joined_at: Time.zone.parse("2024-08-18 2:00:00"), last_page_modified_at: Time.zone.parse("2024-08-19 1:00:00"))
        create(:topic_membership, space:, topic: topic_d, member: user, joined_at: Time.zone.parse("2024-08-18 3:00:00"), last_page_modified_at: nil)
      end

      it "ページが編集された順にトピックが取得できること" do
        expect(
          user.last_page_modified_topics.pluck(:name)
        ).to eq(%w[トピックC トピックB トピックD トピックA])
      end
    end
  end

  describe "#last_modified_pages" do
    context "ページが存在するとき" do
      let!(:space) { create(:space) }
      let!(:user_a) { create(:user, space:) }
      let!(:user_b) { create(:user, space:) }
      let!(:topic) { create(:topic, space:) }
      let!(:page_a) { create(:page, space:, topic:, title: "ページA") }
      let!(:page_b) { create(:page, space:, topic:, title: "ページB") }
      let!(:page_c) { create(:page, space:, topic:, title: "ページC") }

      before do
        create(:page_editorship, space:, page: page_a, editor: user_a, last_page_modified_at: Time.zone.parse("2024-08-18 0:00:00"))
        create(:page_editorship, space:, page: page_b, editor: user_b, last_page_modified_at: Time.zone.parse("2024-08-18 1:00:00"))
        create(:page_editorship, space:, page: page_c, editor: user_b, last_page_modified_at: Time.zone.parse("2024-08-18 2:00:00"))
        create(:page_editorship, space:, page: page_c, editor: user_a, last_page_modified_at: Time.zone.parse("2024-08-18 3:00:00"))
      end

      it "最後に編集したページから取得できること" do
        expect(user_a.last_modified_pages.pluck(:title)).to eq(%w[ページC ページA])
      end
    end
  end
end
